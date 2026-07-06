import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import '../bloc/container_bridge_bloc.dart';
import '../widgets/empty_state.dart';
import '../widgets/loading_indicator.dart';
import '../widgets/error_widget.dart' as helix_error;

class ContainerBridgeScreen extends StatefulWidget {
  const ContainerBridgeScreen({super.key});

  @override
  State<ContainerBridgeScreen> createState() => _ContainerBridgeScreenState();
}

class _ContainerBridgeScreenState extends State<ContainerBridgeScreen> {
  @override
  void initState() {
    super.initState();
    context.read<ContainerBridgeBloc>().add(ContainerBridgeListRequested());
  }

  Color _statusColor(String status) {
    return switch (status.toLowerCase()) {
      'running' => Colors.green,
      'stopped' => Colors.red,
      'paused' => Colors.orange,
      'restarting' => Colors.blue,
      _ => Colors.grey,
    };
  }

  IconData _statusIcon(String status) {
    return switch (status.toLowerCase()) {
      'running' => Icons.play_arrow,
      'stopped' => Icons.stop,
      'paused' => Icons.pause,
      'restarting' => Icons.refresh,
      _ => Icons.help,
    };
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Container Bridge'),
      ),
      body: BlocConsumer<ContainerBridgeBloc, ContainerBridgeState>(
        listener: (context, state) {
          if (state is ContainerBridgeActionSuccess) {
            ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(state.message)));
          }
        },
        builder: (context, state) {
          if (state is ContainerBridgeLoading) {
            return const LoadingIndicator();
          }
          if (state is ContainerBridgeError) {
            return helix_error.ErrorWidget(
              message: state.message,
              onRetry: () => context.read<ContainerBridgeBloc>().add(ContainerBridgeListRequested()),
            );
          }
          if (state is ContainerBridgeListLoaded) {
            if (state.containers.isEmpty) {
              return const EmptyState(message: 'No container bridges configured');
            }
            return ListView.builder(
              itemCount: state.containers.length,
              itemBuilder: (context, index) {
                final container = state.containers[index];
                final status = container['status'] as String? ?? 'unknown';
                return Card(
                  margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
                  child: ListTile(
                    leading: CircleAvatar(
                      backgroundColor: _statusColor(status).withOpacity(0.2),
                      child: Icon(
                        _statusIcon(status),
                        color: _statusColor(status),
                      ),
                    ),
                    title: Text(container['name'] as String? ?? 'Unknown'),
                    subtitle: Text('Image: ${container['image'] ?? 'N/A'}'),
                    trailing: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Chip(
                          label: Text(status),
                          backgroundColor: _statusColor(status).withOpacity(0.2),
                          side: BorderSide(color: _statusColor(status)),
                        ),
                        const SizedBox(width: 8),
                        FilledButton.tonal(
                          onPressed: () {
                            context.read<ContainerBridgeBloc>().add(
                              ContainerBridgeConnect(container['id'] as String),
                            );
                          },
                          child: const Text('Connect'),
                        ),
                        PopupMenuButton<String>(
                          onSelected: (value) {
                            final id = container['id'] as String;
                            switch (value) {
                              case 'start':
                                context.read<ContainerBridgeBloc>().add(ContainerBridgeStart(id));
                                break;
                              case 'stop':
                                context.read<ContainerBridgeBloc>().add(ContainerBridgeStop(id));
                                break;
                              case 'restart':
                                context.read<ContainerBridgeBloc>().add(ContainerBridgeRestart(id));
                                break;
                              case 'logs':
                                context.read<ContainerBridgeBloc>().add(ContainerBridgeLogs(id));
                                break;
                              case 'remove':
                                context.read<ContainerBridgeBloc>().add(ContainerBridgeRemove(id));
                                break;
                            }
                          },
                          itemBuilder: (context) => [
                            const PopupMenuItem(value: 'start', child: Text('Start')),
                            const PopupMenuItem(value: 'stop', child: Text('Stop')),
                            const PopupMenuItem(value: 'restart', child: Text('Restart')),
                            const PopupMenuItem(value: 'logs', child: Text('View Logs')),
                            const PopupMenuItem(value: 'remove', child: Text('Remove')),
                          ],
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
      floatingActionButton: FloatingActionButton(
        onPressed: () {
          _showCreateDialog();
        },
        child: const Icon(Icons.add),
      ),
    );
  }

  void _showCreateDialog() {
    final nameController = TextEditingController();
    final imageController = TextEditingController();
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Create Container Bridge'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              controller: nameController,
              decoration: const InputDecoration(labelText: 'Name'),
            ),
            TextField(
              controller: imageController,
              decoration: const InputDecoration(labelText: 'Image'),
            ),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context), child: const Text('Cancel')),
          FilledButton(
            onPressed: () {
              context.read<ContainerBridgeBloc>().add(ContainerBridgeCreate(
                name: nameController.text,
                image: imageController.text,
              ));
              Navigator.pop(context);
            },
            child: const Text('Create'),
          ),
        ],
      ),
    );
  }
}
