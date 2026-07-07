import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../bloc/workspace_bloc.dart';
import '../models/workspace.dart';
import '../widgets/empty_state.dart';
import '../widgets/workspace_card.dart';
import 'workspace_detail_screen.dart';

/// Workspace list screen with cards, member count, create button, and search.
class WorkspaceListScreen extends StatefulWidget {
  const WorkspaceListScreen({super.key});

  @override
  State<WorkspaceListScreen> createState() => _WorkspaceListScreenState();
}

class _WorkspaceListScreenState extends State<WorkspaceListScreen> {
  final TextEditingController _searchController = TextEditingController();
  String _searchQuery = '';

  @override
  void initState() {
    super.initState();
    context.read<WorkspaceBloc>().add(WorkspaceLoadRequested());
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  List<Workspace> _filterWorkspaces(List<Workspace> workspaces) {
    if (_searchQuery.isEmpty) return workspaces;
    final query = _searchQuery.toLowerCase();
    return workspaces.where((w) {
      return w.name.toLowerCase().contains(query) ||
          (w.description?.toLowerCase().contains(query) ?? false);
    }).toList();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Workspaces'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: 'Refresh',
            onPressed: () {
              context.read<WorkspaceBloc>().add(WorkspaceLoadRequested());
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
              hintText: 'Search workspaces...',
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
          Expanded(
            child: BlocConsumer<WorkspaceBloc, WorkspaceState>(
              listener: (context, state) {
                if (state is WorkspaceError) {
                  ScaffoldMessenger.of(context).showSnackBar(
                    SnackBar(content: Text(state.message)),
                  );
                }
                if (state is WorkspaceOperationSuccess) {
                  ScaffoldMessenger.of(context).showSnackBar(
                    SnackBar(content: Text(state.message)),
                  );
                }
              },
              builder: (context, state) {
                if (state is WorkspaceLoading) {
                  return const Center(child: CircularProgressIndicator());
                }

                if (state is WorkspaceLoaded) {
                  final filtered = _filterWorkspaces(state.workspaces);
                  if (filtered.isEmpty) {
                    return const EmptyState(message: 'No workspaces found');
                  }
                  return ListView.builder(
                    padding: const EdgeInsets.symmetric(horizontal: 16),
                    itemCount: filtered.length,
                    itemBuilder: (context, index) {
                      final workspace = filtered[index];
                      return Padding(
                        padding: const EdgeInsets.only(bottom: 12),
                        child: WorkspaceCard(
                          name: workspace.name,
                          hostCount: workspace.hostIds.length,
                          memberCount: workspace.memberIds.length,
                          onTap: () {
                            Navigator.of(context).push(
                              MaterialPageRoute(
                                builder: (_) => BlocProvider.value(
                                  value: context.read<WorkspaceBloc>(),
                                  child: WorkspaceDetailScreen(workspace: workspace),
                                ),
                              ),
                            );
                          },
                        ),
                      );
                    },
                  );
                }

                return const EmptyState(message: 'No workspaces loaded');
              },
            ),
          ),
        ],
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => _showCreateDialog(context),
        icon: const Icon(Icons.add),
        label: const Text('Create'),
      ),
    );
  }

  void _showCreateDialog(BuildContext context) {
    final nameController = TextEditingController();
    final descController = TextEditingController();
    final formKey = GlobalKey<FormState>();

    showDialog(
      context: context,
      builder: (dialogContext) {
        return AlertDialog(
          title: const Text('Create Workspace'),
          content: Form(
            key: formKey,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                TextFormField(
                  controller: nameController,
                  decoration: const InputDecoration(labelText: 'Name'),
                  validator: (value) {
                    if (value == null || value.trim().isEmpty) {
                      return 'Name is required';
                    }
                    if (value.trim().length < 2) {
                      return 'Name must be at least 2 characters';
                    }
                    return null;
                  },
                ),
                const SizedBox(height: 12),
                TextFormField(
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
                if (!formKey.currentState!.validate()) return;
                final name = nameController.text.trim();
                context.read<WorkspaceBloc>().add(
                  WorkspaceCreateRequested(
                    name: name,
                    description: descController.text.trim(),
                  ),
                );
                Navigator.of(dialogContext).pop();
              },
              child: const Text('Create'),
            ),
          ],
        );
      },
    );
  }
}
