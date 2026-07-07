import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import '../bloc/host_bloc.dart';
import '../services/host_service.dart';
import '../models/host.dart';
import 'host_create_screen.dart';
import 'terminal_screen.dart';
import 'sftp_browser_screen.dart';
import 'port_forward_screen.dart';
import 'recording_list_screen.dart';

class HostDetailScreen extends StatefulWidget {
  final String hostId;

  const HostDetailScreen({super.key, required this.hostId});

  @override
  State<HostDetailScreen> createState() => _HostDetailScreenState();
}

class _HostDetailScreenState extends State<HostDetailScreen> with SingleTickerProviderStateMixin {
  late TabController _tabController;
  Host? _host;
  bool _isLoading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 5, vsync: this);
    _loadHost();
  }

  @override
  void dispose() {
    _tabController.dispose();
    super.dispose();
  }

  Future<void> _loadHost() async {
    try {
      final hostService = context.read<HostService>();
      final host = await hostService.getHostById(widget.hostId);
      if (mounted) {
        setState(() {
          _host = host;
          _isLoading = false;
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          _error = e.toString();
          _isLoading = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(
        title: Text(_host?.name ?? 'Host Detail'),
        bottom: _host != null
            ? TabBar(
                controller: _tabController,
                isScrollable: true,
                tabs: const [
                  Tab(icon: Icon(Icons.info_outline), text: 'Info'),
                  Tab(icon: Icon(Icons.terminal), text: 'Terminal'),
                  Tab(icon: Icon(Icons.folder_open), text: 'SFTP'),
                  Tab(icon: Icon(Icons.route), text: 'Port Forwards'),
                  Tab(icon: Icon(Icons.videocam), text: 'Recordings'),
                ],
              )
            : null,
      ),
      body: _buildBody(theme),
    );
  }

  Widget _buildBody(ThemeData theme) {
    if (_isLoading) {
      return const Center(child: CircularProgressIndicator());
    }

    if (_error != null) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(Icons.error_outline, size: 64, color: theme.colorScheme.error.withOpacity(0.5)),
            const SizedBox(height: 16),
            Text('Failed to load host', style: theme.textTheme.titleMedium),
            const SizedBox(height: 8),
            Text(_error!, style: theme.textTheme.bodySmall, textAlign: TextAlign.center),
            const SizedBox(height: 16),
            ElevatedButton(onPressed: _loadHost, child: const Text('Retry')),
          ],
        ),
      );
    }

    if (_host == null) {
      return const Center(child: Text('Host not found'));
    }

    return TabBarView(
      controller: _tabController,
      children: [
        _InfoTab(host: _host!),
        TerminalScreen(hostId: widget.hostId, hostName: _host!.name),
        const SftpBrowserScreen(),
        const PortForwardScreen(),
        const RecordingListScreen(),
      ],
    );
  }
}

class _InfoTab extends StatelessWidget {
  final Host host;
  const _InfoTab({required this.host});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isDesktop = MediaQuery.of(context).size.width > 800;

    return SingleChildScrollView(
      padding: const EdgeInsets.all(16.0),
      child: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 800),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Card(
                elevation: 2,
                child: Padding(
                  padding: const EdgeInsets.all(20.0),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        children: [
                          CircleAvatar(
                            radius: 32,
                            backgroundColor: _statusColor(host.status).withOpacity(0.15),
                            child: Text(
                              host.name.isNotEmpty ? host.name[0].toUpperCase() : '?',
                              style: TextStyle(
                                fontSize: 28,
                                fontWeight: FontWeight.bold,
                                color: _statusColor(host.status),
                              ),
                            ),
                          ),
                          const SizedBox(width: 16),
                          Expanded(
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Text(host.name, style: theme.textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.bold)),
                                const SizedBox(height: 4),
                                Row(
                                  children: [
                                    Container(
                                      width: 10,
                                      height: 10,
                                      decoration: BoxDecoration(
                                        color: _statusColor(host.status),
                                        shape: BoxShape.circle,
                                      ),
                                    ),
                                    const SizedBox(width: 8),
                                    Text(
                                      host.status.toUpperCase(),
                                      style: theme.textTheme.bodyMedium?.copyWith(
                                            color: _statusColor(host.status),
                                            fontWeight: FontWeight.w600,
                                          ),
                                    ),
                                  ],
                                ),
                              ],
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 20),
                      const Divider(),
                      const SizedBox(height: 12),
                      _InfoRow(icon: Icons.computer, label: 'Hostname', value: host.address),
                      _InfoRow(icon: Icons.numbers, label: 'Port', value: host.port.toString()),
                      _InfoRow(icon: Icons.person, label: 'Username', value: host.username ?? 'Not set'),
                      _InfoRow(icon: Icons.lock, label: 'Auth Method', value: host.authMethod),
                      if (host.organizationId != null)
                        _InfoRow(icon: Icons.business, label: 'Organization', value: host.organizationId!),
                      const SizedBox(height: 12),
                      if (host.tags.isNotEmpty) ...[
                        const Divider(),
                        const SizedBox(height: 12),
                        Text('Tags', style: theme.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.bold)),
                        const SizedBox(height: 8),
                        Wrap(
                          spacing: 8,
                          children: host.tags
                              .map((tag) => Chip(
                                    label: Text(tag),
                                    avatar: const Icon(Icons.label, size: 16),
                                  ))
                              .toList(),
                        ),
                      ],
                    ],
                  ),
                ),
              ),
              const SizedBox(height: 16),
              if (isDesktop)
                Row(
                  children: [
                    Expanded(
                      child: _ActionButton(
                        icon: Icons.terminal,
                        label: 'Connect',
                        color: theme.colorScheme.primary,
                        onTap: () => Navigator.of(context).push(
                          MaterialPageRoute(builder: (_) => TerminalScreen(hostId: widget.hostId, hostName: _host!.name)),
                        ),
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: _ActionButton(
                        icon: Icons.edit,
                        label: 'Edit',
                        color: theme.colorScheme.secondary,
                        onTap: () => Navigator.of(context).push(
                          MaterialPageRoute(builder: (_) => HostCreateScreen(host: host)),
                        ),
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: _ActionButton(
                        icon: Icons.share,
                        label: 'Share',
                        color: theme.colorScheme.tertiary,
                        onTap: () => _showShareSheet(context),
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: _ActionButton(
                        icon: Icons.delete,
                        label: 'Delete',
                        color: Colors.red,
                        onTap: () => _confirmDelete(context),
                      ),
                    ),
                  ],
                )
              else
                Column(
                  children: [
                    _ActionButton(
                      icon: Icons.terminal,
                      label: 'Connect to Host',
                      color: theme.colorScheme.primary,
                      onTap: () => Navigator.of(context).push(
                        MaterialPageRoute(builder: (_) => TerminalScreen(hostId: widget.hostId, hostName: _host!.name)),
                      ),
                    ),
                    const SizedBox(height: 12),
                    Row(
                      children: [
                        Expanded(
                          child: _ActionButton(
                            icon: Icons.edit,
                            label: 'Edit',
                            color: theme.colorScheme.secondary,
                            onTap: () => Navigator.of(context).push(
                              MaterialPageRoute(builder: (_) => HostCreateScreen(host: host)),
                            ),
                          ),
                        ),
                        const SizedBox(width: 12),
                        Expanded(
                          child: _ActionButton(
                            icon: Icons.share,
                            label: 'Share',
                            color: theme.colorScheme.tertiary,
                            onTap: () => _showShareSheet(context),
                          ),
                        ),
                        const SizedBox(width: 12),
                        Expanded(
                          child: _ActionButton(
                            icon: Icons.delete,
                            label: 'Delete',
                            color: Colors.red,
                            onTap: () => _confirmDelete(context),
                          ),
                        ),
                      ],
                    ),
                  ],
                ),
            ],
          ),
        ),
      ),
    );
  }

  void _showShareSheet(BuildContext context) {
    showModalBottomSheet(
      context: context,
      builder: (_) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              leading: const Icon(Icons.copy),
              title: const Text('Copy connection string'),
              onTap: () => Navigator.pop(context),
            ),
            ListTile(
              leading: const Icon(Icons.email),
              title: const Text('Send via email'),
              onTap: () => Navigator.pop(context),
            ),
            ListTile(
              leading: const Icon(Icons.qr_code),
              title: const Text('Show QR code'),
              onTap: () => Navigator.pop(context),
            ),
          ],
        ),
      ),
    );
  }

  void _confirmDelete(BuildContext context) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Delete Host'),
        content: Text('Are you sure you want to delete "${host.name}"? This action cannot be undone.'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Cancel')),
          TextButton(
            onPressed: () {
              Navigator.pop(ctx);
              context.read<HostBloc>().add(HostDeleteRequested(host.id));
              Navigator.of(context).pop();
            },
            child: const Text('Delete', style: TextStyle(color: Colors.red)),
          ),
        ],
      ),
    );
  }

  Color _statusColor(String status) {
    switch (status) {
      case 'online':
        return Colors.green;
      case 'offline':
        return Colors.red;
      case 'connecting':
        return Colors.orange;
      default:
        return Colors.grey;
    }
  }
}

class _InfoRow extends StatelessWidget {
  final IconData icon;
  final String label;
  final String value;

  const _InfoRow({required this.icon, required this.label, required this.value});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8.0),
      child: Row(
        children: [
          Icon(icon, size: 20, color: Theme.of(context).colorScheme.onSurfaceVariant),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(label, style: Theme.of(context).textTheme.bodySmall?.copyWith(color: Theme.of(context).colorScheme.onSurfaceVariant)),
                Text(value, style: Theme.of(context).textTheme.bodyLarge?.copyWith(fontWeight: FontWeight.w500)),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _ActionButton extends StatelessWidget {
  final IconData icon;
  final String label;
  final Color color;
  final VoidCallback onTap;

  const _ActionButton({required this.icon, required this.label, required this.color, required this.onTap});

  @override
  Widget build(BuildContext context) {
    return Card(
      elevation: 2,
      color: color.withOpacity(0.1),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.symmetric(vertical: 16.0, horizontal: 12.0),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(icon, color: color),
              const SizedBox(height: 8),
              Text(label, style: TextStyle(color: color, fontWeight: FontWeight.w600)),
            ],
          ),
        ),
      ),
    );
  }
}
