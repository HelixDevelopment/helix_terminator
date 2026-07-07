import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../bloc/vault_bloc.dart';
import '../models/secret.dart';

/// Secret detail screen with copy buttons, reveal toggle, and edit/delete actions.
class VaultDetailScreen extends StatefulWidget {
  final Secret secret;

  const VaultDetailScreen({super.key, required this.secret});

  @override
  State<VaultDetailScreen> createState() => _VaultDetailScreenState();
}

class _VaultDetailScreenState extends State<VaultDetailScreen> {
  bool _isRevealed = false;
  bool _isEditing = false;

  late TextEditingController _nameController;
  late TextEditingController _valueController;
  late TextEditingController _descController;
  late String _selectedType;

  @override
  void initState() {
    super.initState();
    _nameController = TextEditingController(text: widget.secret.name);
    _valueController = TextEditingController(text: widget.secret.value ?? '••••••••••••');
    _descController = TextEditingController(text: widget.secret.description ?? '');
    _selectedType = widget.secret.type;
  }

  @override
  void dispose() {
    _nameController.dispose();
    _valueController.dispose();
    _descController.dispose();
    super.dispose();
  }

  void _copyToClipboard(String text, String label) {
    Clipboard.setData(ClipboardData(text: text));
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text('$label copied to clipboard')),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.secret.name),
        actions: [
          if (!_isEditing)
            IconButton(
              icon: const Icon(Icons.edit),
              tooltip: 'Edit',
              onPressed: () => setState(() => _isEditing = true),
            ),
          IconButton(
            icon: const Icon(Icons.delete_forever),
            tooltip: 'Delete',
            onPressed: () => _confirmDelete(context),
          ),
        ],
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    _buildInfoRow('ID', widget.secret.id),
                    const Divider(),
                    _buildInfoRow('Type', widget.secret.type.toUpperCase()),
                    const Divider(),
                    _buildInfoRow('Category', widget.secret.category),
                    const Divider(),
                    _buildInfoRow(
                      'Created',
                      widget.secret.createdAt.toLocal().toString(),
                    ),
                    if (widget.secret.updatedAt != null) ...[
                      const Divider(),
                      _buildInfoRow(
                        'Updated',
                        widget.secret.updatedAt!.toLocal().toString(),
                      ),
                    ],
                  ],
                ),
              ),
            ),
            const SizedBox(height: 16),
            if (_isEditing) ...[
              TextField(
                controller: _nameController,
                decoration: const InputDecoration(labelText: 'Name'),
              ),
              const SizedBox(height: 12),
              DropdownButtonFormField<String>(
                value: _selectedType,
                decoration: const InputDecoration(labelText: 'Type'),
                items: const [
                  DropdownMenuItem(value: 'password', child: Text('Password')),
                  DropdownMenuItem(value: 'key', child: Text('Key')),
                  DropdownMenuItem(value: 'token', child: Text('Token')),
                ],
                onChanged: (value) {
                  if (value != null) setState(() => _selectedType = value);
                },
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _valueController,
                decoration: const InputDecoration(labelText: 'Value'),
                obscureText: !_isRevealed,
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _descController,
                decoration: const InputDecoration(labelText: 'Description'),
                maxLines: 3,
              ),
              const SizedBox(height: 24),
              Row(
                children: [
                  Expanded(
                    child: OutlinedButton(
                      onPressed: () => setState(() => _isEditing = false),
                      child: const Text('Cancel'),
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: FilledButton.icon(
                      onPressed: () {
                        context.read<VaultBloc>().add(
                          VaultUpdateRequested(
                            id: widget.secret.id,
                            name: _nameController.text.trim(),
                            value: _valueController.text,
                            type: _selectedType,
                            description: _descController.text.trim(),
                          ),
                        );
                        setState(() => _isEditing = false);
                      },
                      icon: const Icon(Icons.save),
                      label: const Text('Save'),
                    ),
                  ),
                ],
              ),
            ] else ...[
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(16),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        children: [
                          Expanded(
                            child: Text(
                              'Secret Value',
                              style: Theme.of(context).textTheme.titleMedium,
                            ),
                          ),
                          IconButton(
                            icon: Icon(_isRevealed ? Icons.visibility_off : Icons.visibility),
                            tooltip: _isRevealed ? 'Hide' : 'Reveal',
                            onPressed: () {
                              setState(() => _isRevealed = !_isRevealed);
                            },
                          ),
                          IconButton(
                            icon: const Icon(Icons.copy),
                            tooltip: 'Copy value',
                            onPressed: () => _copyToClipboard(widget.secret.value ?? '', 'Value'),
                          ),
                        ],
                      ),
                      const SizedBox(height: 8),
                      Container(
                        width: double.infinity,
                        padding: const EdgeInsets.all(12),
                        decoration: BoxDecoration(
                          color: Theme.of(context).colorScheme.surfaceContainerHighest,
                          borderRadius: BorderRadius.circular(8),
                        ),
                        child: SelectableText(
                          _isRevealed ? (widget.secret.value ?? '••••••••••••••••••') : '••••••••••••••••••',
                          style: TextStyle(
                            fontFamily: 'monospace',
                            color: _isRevealed
                                ? Theme.of(context).colorScheme.onSurface
                                : Theme.of(context).colorScheme.onSurface.withOpacity(0.4),
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
              ),
              const SizedBox(height: 16),
              if (widget.secret.description != null && widget.secret.description!.isNotEmpty)
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          'Description',
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                        const SizedBox(height: 8),
                        Text(widget.secret.description!),
                      ],
                    ),
                  ),
                ),
            ],
          ],
        ),
      ),
    );
  }

  Widget _buildInfoRow(String label, String value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 100,
            child: Text(
              label,
              style: TextStyle(
                color: Theme.of(context).colorScheme.onSurface.withOpacity(0.6),
                fontSize: 14,
              ),
            ),
          ),
          Expanded(
            child: SelectableText(
              value,
              style: const TextStyle(fontSize: 14),
            ),
          ),
        ],
      ),
    );
  }

  void _confirmDelete(BuildContext context) {
    showDialog(
      context: context,
      builder: (dialogContext) {
        return AlertDialog(
          title: const Text('Delete Secret?'),
          content: const Text('This secret will be permanently removed.'),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(dialogContext).pop(),
              child: const Text('Cancel'),
            ),
            FilledButton(
              onPressed: () {
                context.read<VaultBloc>().add(VaultDeleteRequested(widget.secret.id));
                Navigator.of(dialogContext).pop();
                Navigator.of(context).maybePop();
              },
              style: FilledButton.styleFrom(backgroundColor: Colors.red),
              child: const Text('Delete'),
            ),
          ],
        );
      },
    );
  }
}
