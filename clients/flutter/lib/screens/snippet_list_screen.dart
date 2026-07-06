import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import '../bloc/snippet_bloc.dart';
import '../widgets/empty_state.dart';
import '../widgets/loading_indicator.dart';
import '../widgets/error_widget.dart' as helix_error;
import '../screens/snippet_detail_screen.dart';

class SnippetListScreen extends StatefulWidget {
  const SnippetListScreen({super.key});

  @override
  State<SnippetListScreen> createState() => _SnippetListScreenState();
}

class _SnippetListScreenState extends State<SnippetListScreen> {
  final TextEditingController _searchController = TextEditingController();

  @override
  void initState() {
    super.initState();
    context.read<SnippetBloc>().add(SnippetListRequested());
  }

  @override
  void dispose() {
    _searchController.dispose();
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

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Snippets'),
      ),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(16.0),
            child: TextField(
              controller: _searchController,
              decoration: InputDecoration(
                hintText: 'Search snippets...',
                prefixIcon: const Icon(Icons.search),
                suffixIcon: _searchController.text.isNotEmpty
                    ? IconButton(
                        icon: const Icon(Icons.clear),
                        onPressed: () {
                          _searchController.clear();
                          context.read<SnippetBloc>().add(SnippetSearchChanged(''));
                        },
                      )
                    : null,
              ),
              onChanged: (value) {
                context.read<SnippetBloc>().add(SnippetSearchChanged(value));
              },
            ),
          ),
          Expanded(
            child: BlocConsumer<SnippetBloc, SnippetState>(
              listener: (context, state) {
                if (state is SnippetActionSuccess) {
                  ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(state.message)));
                }
              },
              builder: (context, state) {
                if (state is SnippetLoading) {
                  return const LoadingIndicator();
                }
                if (state is SnippetError) {
                  return helix_error.ErrorWidget(
                    message: state.message,
                    onRetry: () => context.read<SnippetBloc>().add(SnippetListRequested()),
                  );
                }
                if (state is SnippetListLoaded) {
                  final snippets = state.snippets.where((s) {
                    if (state.searchQuery.isEmpty) return true;
                    final q = state.searchQuery.toLowerCase();
                    return s.title.toLowerCase().contains(q) ||
                        s.language.toLowerCase().contains(q) ||
                        s.content.toLowerCase().contains(q);
                  }).toList();

                  if (snippets.isEmpty) {
                    return const EmptyState(message: 'No snippets found');
                  }

                  return ListView.builder(
                    itemCount: snippets.length,
                    itemBuilder: (context, index) {
                      final snippet = snippets[index];
                      return Card(
                        margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
                        child: ListTile(
                          leading: Chip(
                            label: Text(snippet.language),
                            backgroundColor: _languageColor(snippet.language).withOpacity(0.2),
                            side: BorderSide(color: _languageColor(snippet.language)),
                          ),
                          title: Text(snippet.title),
                          subtitle: Text(
                            snippet.content.length > 60
                                ? '${snippet.content.substring(0, 60)}...'
                                : snippet.content,
                            style: Theme.of(context).textTheme.bodySmall,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                          ),
                          trailing: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              IconButton(
                                icon: const Icon(Icons.copy),
                                tooltip: 'Copy',
                                onPressed: () {
                                  ScaffoldMessenger.of(context).showSnackBar(
                                    const SnackBar(content: Text('Copied to clipboard')),
                                  );
                                },
                              ),
                              IconButton(
                                icon: const Icon(Icons.delete_outline),
                                tooltip: 'Delete',
                                onPressed: () {
                                  context.read<SnippetBloc>().add(SnippetDelete(snippet.id));
                                },
                              ),
                            ],
                          ),
                          onTap: () {
                            Navigator.push(
                              context,
                              MaterialPageRoute(
                                builder: (_) => SnippetDetailScreen(snippetId: snippet.id),
                              ),
                            );
                          },
                        ),
                      );
                    },
                  );
                }
                return const SizedBox.shrink();
              },
            ),
          ),
        ],
      ),
      floatingActionButton: FloatingActionButton(
        onPressed: () {
          Navigator.push(
            context,
            MaterialPageRoute(
              builder: (_) => const SnippetDetailScreen(snippetId: null),
            ),
          );
        },
        child: const Icon(Icons.add),
      ),
    );
  }
}
