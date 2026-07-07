import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import '../bloc/host_bloc.dart';
import '../models/host.dart';
import 'host_list_screen.dart';
import 'host_create_screen.dart';
import 'terminal_screen.dart';
import 'search_screen.dart';
import 'notification_screen.dart';

class DashboardScreen extends StatefulWidget {
  const DashboardScreen({super.key});

  @override
  State<DashboardScreen> createState() => _DashboardScreenState();
}

class _DashboardScreenState extends State<DashboardScreen> {
  int _currentIndex = 0;

  final List<Widget> _pages = const [
    _DashboardBody(),
    HostListScreen(),
    SearchScreen(),
    NotificationScreen(),
  ];

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: _pages[_currentIndex],
      bottomNavigationBar: NavigationBar(
        selectedIndex: _currentIndex,
        onDestinationSelected: (index) => setState(() => _currentIndex = index),
        destinations: const [
          NavigationDestination(icon: Icon(Icons.dashboard_outlined), selectedIcon: Icon(Icons.dashboard), label: 'Home'),
          NavigationDestination(icon: Icon(Icons.dns_outlined), selectedIcon: Icon(Icons.dns), label: 'Hosts'),
          NavigationDestination(icon: Icon(Icons.search_outlined), selectedIcon: Icon(Icons.search), label: 'Search'),
          NavigationDestination(icon: Icon(Icons.notifications_outlined), selectedIcon: Icon(Icons.notifications), label: 'Alerts'),
        ],
      ),
    );
  }
}

class _DashboardBody extends StatefulWidget {
  const _DashboardBody();

  @override
  State<_DashboardBody> createState() => _DashboardBodyState();
}

class _DashboardBodyState extends State<_DashboardBody> {
  @override
  void initState() {
    super.initState();
    context.read<HostBloc>().add(const HostLoadRequested());
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isDesktop = MediaQuery.of(context).size.width > 800;

    return BlocListener<HostBloc, HostState>(
      listener: (context, state) {
        if (state is HostOperationSuccess) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text(state.message)),
          );
        } else if (state is HostError) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text(state.message), backgroundColor: theme.colorScheme.error),
          );
        }
      },
      child: SafeArea(
        child: RefreshIndicator(
          onRefresh: () async {
            context.read<HostBloc>().add(const HostRefreshRequested());
            await context.read<HostBloc>().stream.firstWhere(
                  (s) => s is HostLoaded || s is HostError,
                );
          },
          child: CustomScrollView(
            slivers: [
              SliverToBoxAdapter(
                child: Padding(
                  padding: const EdgeInsets.all(16.0),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text('Welcome back', style: theme.textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.bold)),
                      const SizedBox(height: 4),
                      Text('Manage your hosts and connections', style: theme.textTheme.bodyMedium?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
                      const SizedBox(height: 20),
                      _buildStatsCards(context, isDesktop),
                      const SizedBox(height: 24),
                      Row(
                        mainAxisAlignment: MainAxisAlignment.spaceBetween,
                        children: [
                          Text('Recent Hosts', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
                          TextButton(
                            onPressed: () => Navigator.of(context).push(MaterialPageRoute(builder: (_) => const HostListScreen())),
                            child: const Text('See all'),
                          ),
                        ],
                      ),
                      const SizedBox(height: 8),
                    ],
                  ),
                ),
              ),
              _buildRecentHosts(context),
              const SliverToBoxAdapter(child: SizedBox(height: 24)),
              SliverToBoxAdapter(
                child: Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 16.0),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text('Quick Actions', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
                      const SizedBox(height: 12),
                      _buildQuickActions(context),
                    ],
                  ),
                ),
              ),
              const SliverToBoxAdapter(child: SizedBox(height: 32)),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildStatsCards(BuildContext context, bool isDesktop) {
    return BlocBuilder<HostBloc, HostState>(
      builder: (context, state) {
        int hostCount = 0;
        int activeSessions = 0;
        int notifications = 0;

        if (state is HostLoaded) {
          hostCount = state.hosts.length;
        } else if (state is HostLoading && state.previousHosts != null) {
          hostCount = state.previousHosts!.length;
        } else if (state is HostError && state.previousHosts != null) {
          hostCount = state.previousHosts!.length;
        }

        final cards = [
          _StatCard(
            icon: Icons.dns,
            label: 'Hosts',
            value: hostCount.toString(),
            color: Colors.indigo,
          ),
          _StatCard(
            icon: Icons.terminal,
            label: 'Active Sessions',
            value: activeSessions.toString(),
            color: Colors.teal,
          ),
          _StatCard(
            icon: Icons.notifications,
            label: 'Notifications',
            value: notifications.toString(),
            color: Colors.orange,
          ),
        ];

        if (isDesktop) {
          return Row(
            children: cards.map((c) => Expanded(child: Padding(padding: const EdgeInsets.symmetric(horizontal: 8), child: c))).toList(),
          );
        }

        return Row(
          children: cards.map((c) => Expanded(child: Padding(padding: const EdgeInsets.symmetric(horizontal: 4), child: c))).toList(),
        );
      },
    );
  }

  Widget _buildRecentHosts(BuildContext context) {
    return BlocBuilder<HostBloc, HostState>(
      builder: (context, state) {
        if (state is HostLoading && state.previousHosts == null) {
          return const SliverToBoxAdapter(
            child: Padding(padding: EdgeInsets.all(16.0), child: Center(child: CircularProgressIndicator())),
          );
        }

        List<Host> hosts = [];
        if (state is HostLoaded) {
          hosts = state.hosts;
        } else if (state is HostLoading && state.previousHosts != null) {
          hosts = state.previousHosts!;
        } else if (state is HostError && state.previousHosts != null) {
          hosts = state.previousHosts!;
        }

        final recentHosts = hosts.take(5).toList();

        if (recentHosts.isEmpty) {
          return const SliverToBoxAdapter(
            child: Padding(
              padding: EdgeInsets.all(16.0),
              child: Center(child: Text('No hosts yet. Add your first host to get started.')),
            ),
          );
        }

        return SliverList(
          delegate: SliverChildBuilderDelegate(
            (context, index) {
              final host = recentHosts[index];
              return Padding(
                padding: const EdgeInsets.symmetric(horizontal: 16.0, vertical: 4.0),
                child: Card(
                  elevation: 2,
                  child: ListTile(
                    leading: CircleAvatar(
                      backgroundColor: _statusColor(host.status).withOpacity(0.15),
                      child: Icon(Icons.computer, color: _statusColor(host.status)),
                    ),
                    title: Text(host.name, style: const TextStyle(fontWeight: FontWeight.w600)),
                    subtitle: Text('${host.address}:${host.port}'),
                    trailing: _StatusIndicator(status: host.status),
                    onTap: () => _showHostOptions(context, host),
                  ),
                ),
              );
            },
            childCount: recentHosts.length,
          ),
        );
      },
    );
  }

  Widget _buildQuickActions(BuildContext context) {
    return Wrap(
      spacing: 12,
      runSpacing: 12,
      children: [
        _QuickActionChip(
          icon: Icons.add,
          label: 'Add Host',
          onTap: () => Navigator.of(context).push(
            MaterialPageRoute(builder: (_) => const HostCreateScreen()),
          ),
        ),
        _QuickActionChip(
          icon: Icons.terminal,
          label: 'Open Terminal',
          onTap: () => Navigator.of(context).push(
            // No specific host is selected from this generic quick action;
            // send the user to pick one, mirroring "Quick Connect" below.
            MaterialPageRoute(builder: (_) => const HostListScreen()),
          ),
        ),
        _QuickActionChip(
          icon: Icons.connect_without_contact,
          label: 'Quick Connect',
          onTap: () => Navigator.of(context).push(
            MaterialPageRoute(builder: (_) => const HostListScreen()),
          ),
        ),
      ],
    );
  }

  void _showHostOptions(BuildContext context, Host host) {
    showModalBottomSheet(
      context: context,
      builder: (_) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              leading: const Icon(Icons.terminal),
              title: const Text('Connect'),
              onTap: () {
                Navigator.pop(context);
                Navigator.of(context).push(
                  MaterialPageRoute(builder: (_) => TerminalScreen(hostId: host.id, hostName: host.name)),
                );
              },
            ),
            ListTile(
              leading: const Icon(Icons.edit),
              title: const Text('Edit'),
              onTap: () {
                Navigator.pop(context);
                Navigator.of(context).push(
                  MaterialPageRoute(builder: (_) => HostCreateScreen(host: host)),
                );
              },
            ),
            ListTile(
              leading: const Icon(Icons.delete, color: Colors.red),
              title: const Text('Delete', style: TextStyle(color: Colors.red)),
              onTap: () {
                Navigator.pop(context);
                context.read<HostBloc>().add(HostDeleteRequested(host.id));
              },
            ),
          ],
        ),
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

class _StatCard extends StatelessWidget {
  final IconData icon;
  final String label;
  final String value;
  final Color color;

  const _StatCard({required this.icon, required this.label, required this.value, required this.color});

  @override
  Widget build(BuildContext context) {
    return Card(
      elevation: 2,
      child: Padding(
        padding: const EdgeInsets.all(16.0),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Icon(icon, color: color, size: 28),
            const SizedBox(height: 12),
            Text(value, style: Theme.of(context).textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.bold)),
            const SizedBox(height: 4),
            Text(label, style: Theme.of(context).textTheme.bodySmall?.copyWith(color: Theme.of(context).colorScheme.onSurfaceVariant)),
          ],
        ),
      ),
    );
  }
}

class _StatusIndicator extends StatelessWidget {
  final String status;
  const _StatusIndicator({required this.status});

  @override
  Widget build(BuildContext context) {
    Color color;
    switch (status) {
      case 'online':
        color = Colors.green;
        break;
      case 'offline':
        color = Colors.red;
        break;
      case 'connecting':
        color = Colors.orange;
        break;
      default:
        color = Colors.grey;
    }
    return Container(
      width: 10,
      height: 10,
      decoration: BoxDecoration(color: color, shape: BoxShape.circle),
    );
  }
}

class _QuickActionChip extends StatelessWidget {
  final IconData icon;
  final String label;
  final VoidCallback onTap;

  const _QuickActionChip({required this.icon, required this.label, required this.onTap});

  @override
  Widget build(BuildContext context) {
    return ActionChip(
      avatar: Icon(icon, size: 18),
      label: Text(label),
      onPressed: onTap,
      elevation: 2,
    );
  }
}
