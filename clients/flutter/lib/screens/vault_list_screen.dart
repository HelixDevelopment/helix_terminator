import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../bloc/vault_bloc.dart';
import '../models/secret.dart';
import '../widgets/empty_state.dart';
import '../widgets/secret_card.dart';
import 'vault_detail_screen.dart';

/// Secret vault list screen with categories, search, and add-secret FAB.
class VaultListScreen extends StatefulWidget {
  const VaultListScreen({super.key});

  @override
  State<VaultListScreen> createState() => _VaultListScreenState();
}

class _VaultListScreenState extends State<VaultListScreen> {
  final TextEditingController _searchController = TextEditingController();
  String _searchQuery = '';
  String? _selectedCategory;

  final List<String> _categories = ['all', 'password', 'key', 'token', 'general'];

  @override
  void initState() {
    super.initState();
    context.read<VaultBloc>().add(VaultLoadRequested());
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  List<Secret> _filterSecrets(List<Secret> secrets) {
    var filtered = secrets;

    if (_selectedCategory != null && _selectedCategory != 'all') {
      filtered = filtered.where((s) => s.category == _selectedCategory).toList();
    }

    if (_searchQuery.isNotEmpty) {
      final query = _searchQuery.toLowerCase();
      filtered = filtered.where((s) {
        return s.name.toLowerCase().contains(query) ||
            s.type.toLowerCase().contains(query) ||
            (s.description?.toLowerCase().contains(query) ?? false);
      }).toList();
    }

    return filtered;
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Vault'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: 'Refresh',
            onPressed: () {
              context.read<VaultBloc>().add(VaultLoadRequested());
            },
          ),
        ],
      ),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(16),
            child: SearchBar(
              controller: _searchController,
              hintText: 'Search secrets...',
              leading: const Icon(Icons.search),
              trailing: [
                if (_searchQuery.isNotEmpty)
                  IconButton(
                    icon: const Icon(Icons.clear),
                    onPressed: () {
                      _searchController.clear();
                      setState(() => _searchQuery = '');
                    },
                  ),
              ],
              onChanged: (value) => setState(() => _searchQuery = value),
            ),
          ),
          SizedBox(
            height: 48,
            child: ListView.builder(
              scrollDirection: Axis.horizontal,
              padding: const EdgeInsets.symmetric(horizontal: 16),
              itemCount: _categories.length,
              itemBuilder: (context, index) {
                final category = _categories[index];
                final isSelected = (_selectedCategory ?? 'all') == category;
                return Padding(
                  padding: const EdgeInsets.only(right: 8),
                  child: ChoiceChip(
                    label: Text(category[0].toUpperCase() + category.substring(1)),
                    selected: isSelected,
                    onSelected: (_) {
                      setState(() {
                        _selectedCategory = category == 'all' ? null : category;
                      });
                    },
                  ),
                );
              },
            ),
          ),
          Expanded(
            child: BlocConsumer<VaultBloc, VaultState>(
              listener: (context, state) {
                if (state is VaultError) {
                  ScaffoldMessenger.of(context).showSnackBar(
                    SnackBar(content: Text(state.message)),
                  );
                }
                if (state is VaultOperationSuccess) {
                  ScaffoldMessenger.of(context).showSnackBar(
                    SnackBar(content: Text(state.message)),
                  );
                }
              },
              builder: (context, state) {
                if (state is VaultLoading) {
                  return const Center(child: CircularProgressIndicator());
                }

                if (state is VaultLoaded) {
                  final filtered = _filterSecrets(state.secrets);
                  if (filtered.isEmpty) {
                    return const EmptyState(message: 'No secrets found');
                  }
                  return ListView.builder(
                    padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
                    itemCount: filtered.length,
                    itemBuilder: (context, index) {
                      final secret = filtered[index];
                      return Padding(
                        padding: const EdgeInsets.only(bottom: 12),
                        child: SecretCard(
                          name: secret.name,
                          type: secret.type,
                          category: secret.category,
                          onTap: () {
                            Navigator.of(context).push(
                              MaterialPageRoute(
                                builder: (_) => BlocProvider.value(
                                  value: context.read<VaultBloc>(),
                                  child: VaultDetailScreen(secret: secret),
                                ),
                              ),
                            );
                          },
                        ),
                      );
                    },
                  );
                }

                return const EmptyState(message: 'No secrets loaded');
              },
            ),
          ),
        ],
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => _showAddSecretDialog(context),
        icon: const Icon(Icons.add),
        label: const Text('Add Secret'),
      ),
    );
  }

  void _showAddSecretDialog(BuildContext context) {
    final nameController = TextEditingController();
    final valueController = TextEditingController();
    final descController = TextEditingController();
    String selectedType = 'password';

    showDialog(
      context: context,
      builder: (dialogContext) {
        return StatefulBuilder(
          builder: (context, setState) {
            return AlertDialog(
              title: const Text('Add Secret'),
              content: SingleChildScrollView(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    TextField(
                      controller: nameController,
                      decoration: const InputDecoration(labelText: 'Name'),
                    ),
                    const SizedBox(height: 12),
                    DropdownButtonFormField<String>(
                      value: selectedType,
                      decoration: const InputDecoration(labelText: 'Type'),
                      items: const [
                        DropdownMenuItem(value: 'password', child: Text('Password')),
                        DropdownMenuItem(value: 'key', child: Text('Key')),
                        DropdownMenuItem(value: 'token', child: Text('Token')),
                      ],
                      onChanged: (value) {
                        if (value != null) setState(() => selectedType = value);
                      },
                    ),
                    const SizedBox(height: 12),
                    TextField(
                      controller: valueController,
                      decoration: const InputDecoration(labelText: 'Value'),
                      obscureText: true,
                    ),
                    const SizedBox(height: 12),
                    TextField(
                      controller: descController,
                      decoration: const InputDecoration(labelText: 'Description'),
                      maxLines: 2,
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
                    final value = valueController.text;
                    if (name.isEmpty || value.isEmpty) return;
                    context.read<VaultBloc>().add(
                      VaultCreateRequested(
                        name: name,
                        value: value,
                        type: selectedType,
                        description: descController.text.trim(),
                      ),
                    );
                    Navigator.of(dialogContext).pop();
                  },
                  child: const Text('Save'),
                ),
              ],
            );
          },
        );
      },
    );
  }
}
