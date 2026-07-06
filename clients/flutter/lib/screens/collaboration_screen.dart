import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import '../bloc/collaboration_bloc.dart';
import '../services/collaboration_service.dart';
import '../widgets/empty_state.dart';
import '../widgets/loading_indicator.dart';
import '../widgets/error_widget.dart' as helix_error;
import '../widgets/user_avatar.dart';

class CollaborationScreen extends StatefulWidget {
  const CollaborationScreen({super.key});

  @override
  State<CollaborationScreen> createState() => _CollaborationScreenState();
}

class _CollaborationScreenState extends State<CollaborationScreen> {
  @override
  void initState() {
    super.initState();
    context.read<CollaborationBloc>().add(CollaborationListRequested());
  }

  void _showCreateDialog() {
    final hostIdController = TextEditingController();
    final nameController = TextEditingController();
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Create Session'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              controller: hostIdController,
              decoration: const InputDecoration(labelText: 'Host ID'),
            ),
            const SizedBox(height: 8),
            TextField(
              controller: nameController,
              decoration: const InputDecoration(labelText: 'Session Name (optional)'),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () {
              context.read<CollaborationBloc>().add(CollaborationCreateSession(
                hostId: hostIdController.text,
                name: nameController.text.isEmpty ? null : nameController.text,
              ));
              Navigator.pop(context);
            },
            child: const Text('Create'),
          ),
        ],
      ),
    );
  }

  void _showJoinDialog() {
    final sessionIdController = TextEditingController();
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Join Session'),
        content: TextField(
          controller: sessionIdController,
          decoration: const InputDecoration(labelText: 'Session ID'),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () {
              context.read<CollaborationBloc>().add(CollaborationJoinSession(sessionIdController.text));
              Navigator.pop(context);
            },
            child: const Text('Join'),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Collaboration'),
      ),
      body: BlocConsumer<CollaborationBloc, CollaborationState>(
        listener: (context, state) {
          if (state is CollaborationActionSuccess) {
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(content: Text(state.message)),
            );
          }
        },
        builder: (context, state) {
          if (state is CollaborationLoading) {
            return const LoadingIndicator();
          }
          if (state is CollaborationError) {
            return helix_error.ErrorWidget(
              message: state.message,
              onRetry: () => context.read<CollaborationBloc>().add(CollaborationListRequested()),
            );
          }
          if (state is CollaborationActive) {
            return _ActiveSessionView(
              session: state.session,
              participants: state.participants,
            );
          }
          if (state is CollaborationListLoaded) {
            if (state.sessions.isEmpty) {
              return EmptyState(
                message: 'No active sessions',
                action: Column(
                  children: [
                    const SizedBox(height: 16),
                    Row(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        FilledButton.icon(
                          onPressed: _showCreateDialog,
                          icon: const Icon(Icons.add),
                          label: const Text('Create'),
                        ),
                        const SizedBox(width: 16),
                        OutlinedButton.icon(
                          onPressed: _showJoinDialog,
                          icon: const Icon(Icons.login),
                          label: const Text('Join'),
                        ),
                      ],
                    ),
                  ],
                ),
              );
            }
            return ListView.builder(
              itemCount: state.sessions.length,
              itemBuilder: (context, index) {
                final session = state.sessions[index];
                return Card(
                  margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
                  child: ListTile(
                    leading: const Icon(Icons.group),
                    title: Text('Session ${session.id.substring(0, 8)}'),
                    subtitle: Text('Host: ${session.hostId}'),
                    trailing: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        IconButton(
                          icon: const Icon(Icons.login),
                          tooltip: 'Join',
                          onPressed: () {
                            context.read<CollaborationBloc>().add(CollaborationJoinSession(session.id));
                          },
                        ),
                        IconButton(
                          icon: const Icon(Icons.delete_outline),
                          tooltip: 'End',
                          onPressed: () {
                            context.read<CollaborationBloc>().add(CollaborationEndSession(session.id));
                          },
                        ),
                      ],
                    ),
                  ),
                );
              },
            );
          }
          return const SizedBox.shrink();
        },
      ),
      floatingActionButton: ExpandableFab(
        icon: const Icon(Icons.add),
        children: [
          ActionButton(
            onPressed: _showCreateDialog,
            icon: const Icon(Icons.create),
            label: 'Create Session',
          ),
          ActionButton(
            onPressed: _showJoinDialog,
            icon: const Icon(Icons.login),
            label: 'Join Session',
          ),
        ],
      ),
    );
  }
}

class _ActiveSessionView extends StatelessWidget {
  final dynamic session;
  final List<Map<String, dynamic>> participants;

  const _ActiveSessionView({required this.session, required this.participants});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Container(
          padding: const EdgeInsets.all(16),
          color: Theme.of(context).colorScheme.primaryContainer,
          child: Row(
            children: [
              Icon(Icons.group, color: Theme.of(context).colorScheme.onPrimaryContainer),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Active Session',
                      style: Theme.of(context).textTheme.titleMedium?.copyWith(
                        color: Theme.of(context).colorScheme.onPrimaryContainer,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                    Text(
                      'ID: ${session.id}',
                      style: Theme.of(context).textTheme.bodySmall?.copyWith(
                        color: Theme.of(context).colorScheme.onPrimaryContainer,
                      ),
                    ),
                  ],
                ),
              ),
              FilledButton.tonal(
                onPressed: () {
                  context.read<CollaborationBloc>().add(CollaborationLeaveSession());
                },
                child: const Text('Leave'),
              ),
            ],
          ),
        ),
        Padding(
          padding: const EdgeInsets.all(16),
          child: Row(
            children: [
              Text(
                'Participants (${participants.length})',
                style: Theme.of(context).textTheme.titleMedium,
              ),
            ],
          ),
        ),
        Expanded(
          child: participants.isEmpty
              ? const EmptyState(message: 'No participants yet')
              : ListView.builder(
                  itemCount: participants.length,
                  itemBuilder: (context, index) {
                    final p = participants[index];
                    return ListTile(
                      leading: UserAvatar(
                        imageUrl: p['avatarUrl'] as String?,
                        initials: (p['name'] as String? ?? 'U').substring(0, 1).toUpperCase(),
                      ),
                      title: Text(p['name'] as String? ?? 'Unknown'),
                      subtitle: Text(p['email'] as String? ?? ''),
                      trailing: p['isHost'] == true
                          ? Chip(
                              label: const Text('Host'),
                              backgroundColor: Theme.of(context).colorScheme.secondaryContainer,
                            )
                          : null,
                    );
                  },
                ),
        ),
      ],
    );
  }
}

class ExpandableFab extends StatelessWidget {
  final Widget icon;
  final List<Widget> children;

  const ExpandableFab({super.key, required this.icon, required this.children});

  @override
  Widget build(BuildContext context) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        ...children,
        const SizedBox(height: 8),
        FloatingActionButton(
          onPressed: () {},
          child: icon,
        ),
      ],
    );
  }
}

class ActionButton extends StatelessWidget {
  final VoidCallback onPressed;
  final Widget icon;
  final String label;

  const ActionButton({super.key, required this.onPressed, required this.icon, required this.label});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
            decoration: BoxDecoration(
              color: Theme.of(context).colorScheme.surface,
              borderRadius: BorderRadius.circular(4),
              boxShadow: [
                BoxShadow(
                  color: Colors.black.withOpacity(0.1),
                  blurRadius: 4,
                ),
              ],
            ),
            child: Text(label, style: Theme.of(context).textTheme.bodySmall),
          ),
          const SizedBox(width: 8),
          FloatingActionButton.small(
            onPressed: onPressed,
            child: icon,
          ),
        ],
      ),
    );
  }
}
