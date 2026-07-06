import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import '../bloc/port_forward_bloc.dart';
import '../widgets/empty_state.dart';
import '../widgets/loading_indicator.dart';
import '../widgets/error_widget.dart' as helix_error;
import '../widgets/status_chip.dart';

class PortForwardScreen extends StatefulWidget {
  const PortForwardScreen({super.key});

  @override
  State<PortForwardScreen> createState() => _PortForwardScreenState();
}

class _PortForwardScreenState extends State<PortForwardScreen> {
  @override
  void initState() {
    super.initState();
    context.read<PortForwardBloc>().add(PortForwardListRequested());
  }

  void _showCreateDialog() {
    final hostIdController = TextEditingController();
    final localPortController = TextEditingController();
    final remotePortController = TextEditingController();
    final remoteHostController = TextEditingController(text: 'localhost');

    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Create Port Forward'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              controller: hostIdController,
              decoration: const InputDecoration(labelText: 'Host ID'),
            ),
            TextField(
              controller: localPortController,
              decoration: const InputDecoration(labelText: 'Local Port'),
              keyboardType: TextInputType.number,
            ),
            TextField(
              controller: remotePortController,
              decoration: const InputDecoration(labelText: 'Remote Port'),
              keyboardType: TextInputType.number,
            ),
            TextField(
              controller: remoteHostController,
              decoration: const InputDecoration(labelText: 'Remote Host'),
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
              final localPort = int.tryParse(localPortController.text) ?? 0;
              final remotePort = int.tryParse(remotePortController.text) ?? 0;
              if (localPort > 0 && remotePort > 0 && hostIdController.text.isNotEmpty) {
                context.read<PortForwardBloc>().add(PortForwardCreate(
                  hostId: hostIdController.text,
                  localPort: localPort,
                  remotePort: remotePort,
                  remoteHost: remoteHostController.text,
                ));
                Navigator.pop(context);
              }
            },
            child: const Text('Create'),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return DefaultTabController(
      length: 2,
      child: Scaffold(
        appBar: AppBar(
          title: const Text('Port Forwarding'),
          bottom: const TabBar(
            tabs: [
              Tab(icon: Icon(Icons.list), text: 'Rules'),
              Tab(icon: Icon(Icons.link), text: 'Active'),
            ],
          ),
        ),
        body: TabBarView(
          children: [
            _RulesTab(onCreate: _showCreateDialog),
            _ActiveConnectionsTab(),
          ],
        ),
        floatingActionButton: FloatingActionButton(
          onPressed: _showCreateDialog,
          child: const Icon(Icons.add),
        ),
      ),
    );
  }
}

class _RulesTab extends StatelessWidget {
  final VoidCallback onCreate;
  const _RulesTab({required this.onCreate});

  @override
  Widget build(BuildContext context) {
    return BlocConsumer<PortForwardBloc, PortForwardState>(
      listener: (context, state) {
        if (state is PortForwardActionSuccess) {
          ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(state.message)));
          context.read<PortForwardBloc>().add(PortForwardListRequested());
        }
      },
      builder: (context, state) {
        if (state is PortForwardLoading) {
          return const LoadingIndicator();
        }
        if (state is PortForwardError) {
          return helix_error.ErrorWidget(
            message: state.message,
            onRetry: () => context.read<PortForwardBloc>().add(PortForwardListRequested()),
          );
        }
        if (state is PortForwardListLoaded) {
          if (state.rules.isEmpty) {
            return EmptyState(
              message: 'No port forward rules',
              action: FilledButton.icon(
                onPressed: onCreate,
                icon: const Icon(Icons.add),
                label: const Text('Create Rule'),
              ),
            );
          }
          return ListView.builder(
            itemCount: state.rules.length,
            itemBuilder: (context, index) {
              final rule = state.rules[index];
              return Card(
                margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
                child: ListTile(
                  leading: Icon(
                    Icons.route,
                    color: rule.active ? Colors.green : Colors.grey,
                  ),
                  title: Text('${rule.localPort} → ${rule.remoteHost}:${rule.remotePort}'),
                  subtitle: Text('Host: ${rule.hostId}'),
                  trailing: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      StatusChip(
                        label: rule.active ? 'Active' : 'Inactive',
                        color: rule.active ? Colors.green : Colors.grey,
                      ),
                      const SizedBox(width: 8),
                      IconButton(
                        icon: Icon(rule.active ? Icons.stop : Icons.play_arrow),
                        tooltip: rule.active ? 'Stop' : 'Start',
                        onPressed: () {
                          if (rule.active) {
                            context.read<PortForwardBloc>().add(PortForwardStop(rule.id));
                          } else {
                            context.read<PortForwardBloc>().add(PortForwardStart(rule.id));
                          }
                        },
                      ),
                      IconButton(
                        icon: const Icon(Icons.delete_outline),
                        tooltip: 'Delete',
                        onPressed: () {
                          context.read<PortForwardBloc>().add(PortForwardDelete(rule.id));
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
    );
  }
}

class _ActiveConnectionsTab extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return BlocBuilder<PortForwardBloc, PortForwardState>(
      builder: (context, state) {
        if (state is PortForwardActiveConnectionsLoaded) {
          if (state.connections.isEmpty) {
            return const EmptyState(message: 'No active connections');
          }
          return ListView.builder(
            itemCount: state.connections.length,
            itemBuilder: (context, index) {
              final conn = state.connections[index];
              return Card(
                margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
                child: ListTile(
                  leading: const Icon(Icons.link, color: Colors.green),
                  title: Text('${conn['localPort']} → ${conn['remoteHost']}:${conn['remotePort']}'),
                  subtitle: Text('Connected: ${conn['connectedAt']}'),
                  trailing: Chip(
                    label: Text('${conn['bytesTransferred'] ?? 0} bytes'),
                  ),
                ),
              );
            },
          );
        }
        if (state is PortForwardLoading) {
          return const LoadingIndicator();
        }
        return const EmptyState(message: 'No active connections');
      },
    );
  }
}
