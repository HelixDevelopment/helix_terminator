import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import '../bloc/snippet_bloc.dart';
import '../widgets/loading_indicator.dart';
import '../widgets/error_widget.dart' as helix_error;

class SnippetDetailScreen extends StatefulWidget {
  final String? snippetId;
  const SnippetDetailScreen({super.key, this.snippetId});

  @override
  State<SnippetDetailScreen> createState() => _SnippetDetailScreenState();
}

class _SnippetDetailScreenState extends State<SnippetDetailScreen> {
  final _titleController = TextEditingController();
  final _contentController = TextEditingController();
  final _languageController = TextEditingController();
  bool _isEditing = false;

  @override
  void initState() {
    super.initState();
    _isEditing = widget.snippetId == null;
    if (widget.snippetId != null) {
      context.read<SnippetBloc>().add(SnippetLoadRequested(widget.snippetId!));
    }
  }

  @override
  void dispose() {
    _titleController.dispose();
    _contentController.dispose();
    _languageController.dispose();
    super.dispose();
  }

  Color _languageColor(String language) {
    return switch (language.toLowerCase()) {
      'dart' => Colors.blue,
      'python' || 'py' => Colors.yellow.shade700,
      'javascript' || 'js' => Colors.orange,
      'typescript' || 'ts' => Colors.blue.shade700,
      'go' => Colors.cyan,
      'rust' || 'rs' => Colors.orange.shade800,
      'java' => Colors.red,
      'cpp' || 'c' => Colors.blue.shade900,
      'bash' || 'sh' => Colors.green,
      'yaml' || 'yml' => Colors.grey,
      'json' => Colors.grey.shade700,
      'sql' => Colors.purple,
      _ => Colors.grey,
    };
  }

  Widget _buildSyntaxHighlightedCode(String code, String language) {
    final color = _languageColor(language);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: Theme.of(context).brightness == Brightness.dark
            ? const Color(0xFF1E293B)
            : const Color(0xFFF1F5F9),
        borderRadius: BorderRadius.circular(8),
      ),
      child: SelectableText(
        code,
        style: TextStyle(
          fontFamily: 'monospace',
          fontSize: 14,
          color: color,
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.snippetId == null ? 'New Snippet' : 'Snippet'),
        actions: [
          if (widget.snippetId != null)
            IconButton(
              icon: Icon(_isEditing ? Icons.save : Icons.edit),
              tooltip: _isEditing ? 'Save' : 'Edit',
              onPressed: () {
                if (_isEditing) {
                  if (_titleController.text.isNotEmpty && _contentController.text.isNotEmpty) {
                    context.read<SnippetBloc>().add(SnippetUpdate(
                      widget.snippetId!,
                      title: _titleController.text,
                      content: _contentController.text,
                      language: _languageController.text,
                    ));
                  }
                }
                setState(() => _isEditing = !_isEditing);
              },
            ),
          if (widget.snippetId != null)
            IconButton(
              icon: const Icon(Icons.delete_outline),
              tooltip: 'Delete',
              onPressed: () {
                showDialog(
                  context: context,
                  builder: (context) => AlertDialog(
                    title: const Text('Delete Snippet'),
                    content: const Text('Are you sure you want to delete this snippet?'),
                    actions: [
                      TextButton(onPressed: () => Navigator.pop(context), child: const Text('Cancel')),
                      FilledButton(
                        onPressed: () {
                          context.read<SnippetBloc>().add(SnippetDelete(widget.snippetId!));
                          Navigator.pop(context);
                          Navigator.pop(context);
                        },
                        child: const Text('Delete'),
                      ),
                    ],
                  ),
                );
              },
            ),
        ],
      ),
      body: BlocConsumer<SnippetBloc, SnippetState>(
        listener: (context, state) {
          if (state is SnippetActionSuccess) {
            ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(state.message)));
          }
          if (state is SnippetDetailLoaded) {
            _titleController.text = state.snippet.title;
            _contentController.text = state.snippet.content;
            _languageController.text = state.snippet.language;
          }
          if (state is SnippetCreated) {
            Navigator.pop(context);
          }
        },
        builder: (context, state) {
          if (state is SnippetLoading && widget.snippetId != null) {
            return const LoadingIndicator();
          }
          if (state is SnippetError) {
            return helix_error.ErrorWidget(
              message: state.message,
              onRetry: () {
                if (widget.snippetId != null) {
                  context.read<SnippetBloc>().add(SnippetLoadRequested(widget.snippetId!));
                }
              },
            );
          }

          return SingleChildScrollView(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                if (_isEditing) ...[
                  TextField(
                    controller: _titleController,
                    decoration: const InputDecoration(labelText: 'Title'),
                  ),
                  const SizedBox(height: 16),
                  TextField(
                    controller: _languageController,
                    decoration: const InputDecoration(labelText: 'Language'),
                  ),
                  const SizedBox(height: 16),
                  TextField(
                    controller: _contentController,
                    decoration: const InputDecoration(
                      labelText: 'Content',
                      alignLabelWithHint: true,
                    ),
                    maxLines: null,
                    minLines: 10,
                  ),
                  if (widget.snippetId == null) ...[
                    const SizedBox(height: 24),
                    FilledButton.icon(
                      onPressed: () {
                        if (_titleController.text.isNotEmpty && _contentController.text.isNotEmpty) {
                          context.read<SnippetBloc>().add(SnippetCreate(
                            title: _titleController.text,
                            content: _contentController.text,
                            language: _languageController.text.isEmpty ? 'text' : _languageController.text,
                          ));
                        }
                      },
                      icon: const Icon(Icons.save),
                      label: const Text('Create Snippet'),
                    ),
                  ],
                ] else ...[
                  if (state is SnippetDetailLoaded) ...[
                    Row(
                      children: [
                        Expanded(
                          child: Text(
                            state.snippet.title,
                            style: Theme.of(context).textTheme.headlineSmall,
                          ),
                        ),
                        Chip(
                          label: Text(state.snippet.language),
                          backgroundColor: _languageColor(state.snippet.language).withOpacity(0.2),
                          side: BorderSide(color: _languageColor(state.snippet.language)),
                        ),
                      ],
                    ),
                    const SizedBox(height: 16),
                    Row(
                      children: [
                        FilledButton.tonal.icon(
                          onPressed: () {
                            Clipboard.setData(ClipboardData(text: state.snippet.content));
                            ScaffoldMessenger.of(context).showSnackBar(
                              const SnackBar(content: Text('Copied to clipboard')),
                            );
                          },
                          icon: const Icon(Icons.copy),
                          label: const Text('Copy'),
                        ),
                      ],
                    ),
                    const SizedBox(height: 16),
                    _buildSyntaxHighlightedCode(state.snippet.content, state.snippet.language),
                  ],
                ],
              ],
            ),
          );
        },
      ),
    );
  }
}
