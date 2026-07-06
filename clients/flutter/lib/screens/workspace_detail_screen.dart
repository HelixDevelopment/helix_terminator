import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../bloc/workspace_bloc.dart';
import '../models/workspace.dart';

/// Workspace detail screen with member list, host list, activity feed, and settings tab.
class WorkspaceDetailScreen extends StatefulWidget {
  final Workspace workspace;

  const WorkspaceDetailScreen({super.key, required this.workspace});

  @override
  State<WorkspaceDetailScreen> createState() => _WorkspaceDetailScreenState();
}

class _WorkspaceDetailScreenState extends State<WorkspaceDetailScreen>
    with SingleTickerProviderStateMixin {
  late TabController _tabController;
  final TextEditingController _memberController = TextEditingController();

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 4, vsync: this);
  }

  @override
  void dispose() {
    _tabController.dispose();
    _memberController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.workspace.name),
        bottom: TabBar(
          controller: _tabController,
          tabs: const [
            Tab(icon: Icon(Icons.people), text: 'Members'),
            Tab(icon: Icon(Icons.dns), text: 'Hosts'),
            Tab(icon: Icon(Icons.timeline), text: 'Activity'),
            Tab(icon: Icon(Icons.settings), text: 'Settings'),
          ],
        ),
      ),
      body: TabBarView(
        controller: _tabController,
        children: [
          _MembersTab(workspace: widget.workspace),
          _HostsTab(workspace: widget.workspace),
          _ActivityTab(workspace: widget.workspace),
          _SettingsTab(workspace: widget.workspace),
        ],
      ),
    );
  }
}

class _MembersTab extends StatelessWidget {
  final Workspace workspace;
  const _MembersTab({required this.workspace});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.all(16),
          child: Row(
            children: [
              Expanded(
                child: TextField(
                  decoration: const InputDecoration(
                    hintText: 'User ID to add...',
                    prefixIcon: Icon(Icons.person_add),
                  ),
                ),
              ),
              const SizedBox(width: 12),
              FilledButton.icon(
                onPressed: () {
                  // In a real app, read the text field value.
                  context.read<WorkspaceBloc>().add(
                    WorkspaceAddMember(
                      workspaceId: workspace.id,
                      userId: 'new-user-id',
                    ),
                  );
                },
                icon: const Icon(Icons.add),
                label: const Text('Add'),
              ),
            ],
          ),
        ),
        Expanded(
          child: workspace.memberIds.isEmpty
              ? const Center(child: Text('No members yet'))
              : ListView.builder(
                  itemCount: workspace.memberIds.length,
                  itemBuilder: (context, index) {
                    final memberId = workspace.memberIds[index];
                    return ListTile(
                      leading: CircleAvatar(
                        child: Text(memberId.substring(0, 1).toUpperCase()),
                      ),
                      title: Text(memberId),
                      subtitle: const Text('Member'),
                      trailing: IconButton(
                        icon: const Icon(Icons.remove_circle_outline),
                        onPressed: () {
                          // Remove member action.
                        },
                      ),
                    );
                  },
                ),
        ),
      ],
    );
  }
}

class _HostsTab extends StatelessWidget {
  final Workspace workspace;
  const _HostsTab({required this.workspace});

  @override
  Widget build(BuildContext context) {
    if (workspace.hostIds.isEmpty) {
      return const Center(child: Text('No hosts in this workspace'));
    }
    return ListView.builder(
      itemCount: workspace.hostIds.length,
      itemBuilder: (context, index) {
        final hostId = workspace.hostIds[index];
        return ListTile(
          leading: const Icon(Icons.computer),
          title: Text(hostId),
          subtitle: const Text('SSH host'),
          trailing: IconButton(
            icon: const Icon(Icons.terminal),
            onPressed: () {
              // Navigate to terminal.
            },
          ),
        );
      },
    );
  }
}

class _ActivityTab extends StatelessWidget {
  final Workspace workspace;
  const _ActivityTab({required this.workspace});

  @override
  Widget build(BuildContext context) {
    final activities = [
      'Workspace created',
      'Host added: prod-server-01',
      'Member joined: alice@example.com',
    ];
    return ListView.builder(
      padding: const EdgeInsets.all(16),
      itemCount: activities.length,
      itemBuilder: (context, index) {
        return Card(
          child: ListTile(
            leading: const Icon(Icons.history),
            title: Text(activities[index]),
            subtitle: Text('${DateTime.now().subtract(Duration(days: index))}'),
          ),
        );
      },
    );
  }
}

class _SettingsTab extends StatefulWidget {
  final Workspace workspace;
  const _SettingsTab({required this.workspace});

  @override
  State<_SettingsTab> createState() => _SettingsTabState();
}

class _SettingsTabState extends State<_SettingsTab> {
  late TextEditingController _nameController;
  late TextEditingController _descController;

  @override
  void initState() {
    super.initState();
    _nameController = TextEditingController(text: widget.workspace.name);
    _descController = TextEditingController(text: widget.workspace.description ?? '');
  }

  @override
  void dispose() {
    _nameController.dispose();
    _descController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          TextField(
            controller: _nameController,
            decoration: const InputDecoration(labelText: 'Workspace Name'),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _descController,
            decoration: const InputDecoration(labelText: 'Description'),
            maxLines: 3,
          ),
          const SizedBox(height: 24),
          FilledButton.icon(
            onPressed: () {
              context.read<WorkspaceBloc>().add(
                WorkspaceUpdateRequested(
                  id: widget.workspace.id,
                  name: _nameController.text.trim(),
                  description: _descController.text.trim(),
                ),
              );
            },
            icon: const Icon(Icons.save),
            label: const Text('Save Changes'),
          ),
          const SizedBox(height: 24),
          OutlinedButton.icon(
            onPressed: () => _confirmDelete(context),
            icon: const Icon(Icons.delete_forever, color: Colors.red),
            label: const Text('Delete Workspace', style: TextStyle(color: Colors.red)),
            style: OutlinedButton.styleFrom(
              side: const BorderSide(color: Colors.red),
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
          title: const Text('Delete Workspace?'),
          content: const Text('This action cannot be undone.'),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(dialogContext).pop(),
              child: const Text('Cancel'),
            ),
            FilledButton(
              onPressed: () {
                context.read<WorkspaceBloc>().add(
                  WorkspaceDeleteRequested(widget.workspace.id),
                );
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
