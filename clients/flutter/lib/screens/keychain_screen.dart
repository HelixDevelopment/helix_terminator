import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../models/ssh_key.dart';
import '../widgets/key_item.dart';
import '../widgets/empty_state.dart';

/// SSH key management screen with key list, generate key, import key, and copy public key.
class KeychainScreen extends StatefulWidget {
  const KeychainScreen({super.key});

  @override
  State<KeychainScreen> createState() => _KeychainScreenState();
}

class _KeychainScreenState extends State<KeychainScreen> {
  final List<SshKey> _keys = [];
  bool _isLoading = false;

  @override
  void initState() {
    super.initState();
    _loadKeys();
  }

  Future<void> _loadKeys() async {
    setState(() => _isLoading = true);
    // In production, call the keychain API.
    await Future.delayed(const Duration(milliseconds: 400));
    setState(() {
      _keys.addAll([
        SshKey(
          id: 'key-1',
          name: 'Personal MacBook',
          fingerprint: 'SHA256:abcd1234efgh5678ijkl9012mnop3456qrst7890',
          createdAt: DateTime.now().subtract(const Duration(days: 30)),
        ),
        SshKey(
          id: 'key-2',
          name: 'Work Laptop',
          fingerprint: 'SHA256:xyz1234abc5678def9012ghi3456jkl7890mnop',
          createdAt: DateTime.now().subtract(const Duration(days: 7)),
        ),
      ]);
      _isLoading = false;
    });
  }

  void _generateKey() {
    showDialog(
      context: context,
      builder: (dialogContext) {
        final nameController = TextEditingController();
        return AlertDialog(
          title: const Text('Generate SSH Key'),
          content: TextField(
            controller: nameController,
            decoration: const InputDecoration(
              labelText: 'Key Name',
              hintText: 'e.g. Production Server Key',
            ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(dialogContext).pop(),
              child: const Text('Cancel'),
            ),
            FilledButton(
              onPressed: () {
                final name = nameController.text.trim();
                if (name.isEmpty) return;
                setState(() {
                  _keys.add(SshKey(
                    id: 'key-${DateTime.now().millisecondsSinceEpoch}',
                    name: name,
                    fingerprint: 'SHA256:${DateTime.now().millisecondsSinceEpoch}',
                    createdAt: DateTime.now(),
                  ));
                });
                Navigator.of(dialogContext).pop();
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(content: Text('SSH key generated')),
                );
              },
              child: const Text('Generate'),
            ),
          ],
        );
      },
    );
  }

  void _importKey() {
    showDialog(
      context: context,
      builder: (dialogContext) {
        final nameController = TextEditingController();
        final keyController = TextEditingController();
        return AlertDialog(
          title: const Text('Import SSH Key'),
          content: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                TextField(
                  controller: nameController,
                  decoration: const InputDecoration(labelText: 'Key Name'),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: keyController,
                  decoration: const InputDecoration(
                    labelText: 'Private Key',
                    hintText: 'Paste PEM-encoded private key...',
                  ),
                  maxLines: 6,
                ),
              ],
            ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(dialogContext).pop(),
              child: const Text('Cancel'),
            ),
            FilledButton(
              onPressed: () {
                final name = nameController.text.trim();
                if (name.isEmpty || keyController.text.isEmpty) return;
                setState(() {
                  _keys.add(SshKey(
                    id: 'key-import-${DateTime.now().millisecondsSinceEpoch}',
                    name: name,
                    fingerprint: 'SHA256:imported${DateTime.now().millisecondsSinceEpoch}',
                    createdAt: DateTime.now(),
                  ));
                });
                Navigator.of(dialogContext).pop();
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(content: Text('SSH key imported')),
                );
              },
              child: const Text('Import'),
            ),
          ],
        );
      },
    );
  }

  void _copyPublicKey(SshKey key) {
    final publicKey = 'ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI... ${key.name}';
    Clipboard.setData(ClipboardData(text: publicKey));
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('Public key copied to clipboard')),
    );
  }

  void _deleteKey(SshKey key) {
    showDialog(
      context: context,
      builder: (dialogContext) {
        return AlertDialog(
          title: const Text('Delete Key?'),
          content: Text('Remove "${key.name}" from your keychain?'),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(dialogContext).pop(),
              child: const Text('Cancel'),
            ),
            FilledButton(
              onPressed: () {
                setState(() => _keys.removeWhere((k) => k.id == key.id));
                Navigator.of(dialogContext).pop();
              },
              style: FilledButton.styleFrom(backgroundColor: Colors.red),
              child: const Text('Delete'),
            ),
          ],
        );
      },
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Keychain'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: 'Refresh',
            onPressed: _loadKeys,
          ),
        ],
      ),
      body: _isLoading
          ? const Center(child: CircularProgressIndicator())
          : _keys.isEmpty
              ? const EmptyState(message: 'No SSH keys in your keychain')
              : ListView.builder(
                  padding: const EdgeInsets.all(16),
                  itemCount: _keys.length,
                  itemBuilder: (context, index) {
                    final key = _keys[index];
                    return Card(
                      margin: const EdgeInsets.only(bottom: 12),
                      child: Column(
                        children: [
                          KeyItem(
                            name: key.name,
                            fingerprint: key.fingerprint,
                            onTap: () {},
                          ),
                          const Divider(height: 1),
                          Padding(
                            padding: const EdgeInsets.symmetric(horizontal: 8),
                            child: Row(
                              children: [
                                TextButton.icon(
                                  icon: const Icon(Icons.copy, size: 18),
                                  label: const Text('Copy Public Key'),
                                  onPressed: () => _copyPublicKey(key),
                                ),
                                const Spacer(),
                                IconButton(
                                  icon: const Icon(Icons.delete_outline, color: Colors.red),
                                  tooltip: 'Delete',
                                  onPressed: () => _deleteKey(key),
                                ),
                              ],
                            ),
                          ),
                        ],
                      ),
                    );
                  },
                ),
      floatingActionButton: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.end,
        children: [
          FloatingActionButton.small(
            heroTag: 'import_key',
            onPressed: _importKey,
            tooltip: 'Import Key',
            child: const Icon(Icons.upload_file),
          ),
          const SizedBox(height: 12),
          FloatingActionButton.extended(
            heroTag: 'generate_key',
            onPressed: _generateKey,
            icon: const Icon(Icons.add),
            label: const Text('Generate Key'),
          ),
        ],
      ),
    );
  }
}
