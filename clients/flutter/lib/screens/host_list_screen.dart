import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import '../bloc/host_bloc.dart';
import '../models/host.dart';
import 'host_detail_screen.dart';
import 'host_create_screen.dart';

class HostListScreen extends StatefulWidget {
  const HostListScreen({super.key});

  @override
  State<HostListScreen> createState() => _HostListScreenState();
}

class _HostListScreenState extends State<HostListScreen> {
  final TextEditingController _searchController = TextEditingController();
  String _searchQuery = '';
  String? _selectedOrgFilter;
  String? _selectedStatusFilter;

  @override
  void initState() {
    super.initState();
    context.read<HostBloc>().add(const HostLoadRequested());
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  List<Host> _filterHosts(List<Host> hosts) {
    return hosts.where((host) {
      final matchesSearch = _searchQuery.isEmpty ||
          host.name.toLowerCase().contains(_searchQuery.toLowerCase()) ||
          host.address.toLowerCase().contains(_searchQuery.toLowerCase()) ||
          host.tags.any((tag) => tag.toLowerCase().contains(_searchQuery.toLowerCase()));
      final matchesOrg = _selectedOrgFilter == null || host.organizationId == _selectedOrgFilter;
      final matchesStatus = _selectedStatusFilter == null || host.status == _selectedStatusFilter;
      return matchesSearch && matchesOrg && matchesStatus;
    }).toList();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Hosts'),
        actions: [
          IconButton(
            icon: const Icon(Icons.filter_list),
            onPressed: _showFilterSheet,
          ),
        ],
      ),
      body: BlocListener<HostBloc, HostState>(
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
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.all(12.0),
              child: SearchBar(
                controller: _searchController,
                hintText: 'Search hosts...',
                leading: const Icon(Icons.search),
                trailing: _searchQuery.isNotEmpty
                    ? [
                        IconButton(
                          icon: const Icon(Icons.clear),
                          onPressed: () {
                            _searchController.clear();
                            setState(() => _searchQuery = '');
                          },
                        ),
                      ]
                    : null,
                onChanged: (value) => setState(() => _searchQuery = value),
                elevation: const WidgetStatePropertyAll(2),
              ),
            ),
            if (_selectedOrgFilter != null || _selectedStatusFilter != null)
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 12.0),
                child: Wrap(
                  spacing: 8,
                  children: [
                    if (_selectedOrgFilter != null)
                      Chip(
                        label: Text('Org: $_selectedOrgFilter'),
                        onDeleted: () => setState(() => _selectedOrgFilter = null),
                        deleteIcon: const Icon(Icons.close, size: 18),
                      ),
                    if (_selectedStatusFilter != null)
                      Chip(
                        label: Text('Status: $_selectedStatusFilter'),
                        onDeleted: () => setState(() => _selectedStatusFilter = null),
                        deleteIcon: const Icon(Icons.close, size: 18),
                      ),
                  ],
                ),
              ),
            Expanded(
              child: BlocBuilder<HostBloc, HostState>(
                builder: (context, state) {
                  if (state is HostLoading && state.previousHosts == null) {
                    return const Center(child: CircularProgressIndicator());
                  }

                  List<Host> hosts = [];
                  if (state is HostLoaded) {
                    hosts = state.hosts;
                  } else if (state is HostLoading && state.previousHosts != null) {
                    hosts = state.previousHosts!;
                  } else if (state is HostError && state.previousHosts != null) {
                    hosts = state.previousHosts!;
                  }

                  final filtered = _filterHosts(hosts);

                  if (filtered.isEmpty && hosts.isEmpty) {
                    return Center(
                      child: Column(
                        mainAxisAlignment: MainAxisAlignment.center,
                        children: [
                          Icon(Icons.dns_outlined, size: 64, color: theme.colorScheme.onSurfaceVariant.withOpacity(0.4)),
                          const SizedBox(height: 16),
                          Text('No hosts found', style: theme.textTheme.titleMedium),
                          const SizedBox(height: 8),
                          ElevatedButton.icon(
                            onPressed: () => _navigateToCreate(context),
                            icon: const Icon(Icons.add),
                            label: const Text('Add Host'),
                          ),
                        ],
                      ),
                    );
                  }

                  if (filtered.isEmpty) {
                    return Center(
                      child: Text('No hosts match your filters', style: theme.textTheme.bodyLarge),
                    );
                  }

                  return RefreshIndicator(
                    onRefresh: () async {
                      context.read<HostBloc>().add(const HostRefreshRequested());
                      await context.read<HostBloc>().stream.firstWhere(
                            (s) => s is HostLoaded || s is HostError,
                          );
                    },
                    child: ListView.builder(
                      padding: const EdgeInsets.only(bottom: 80),
                      itemCount: filtered.length,
                      itemBuilder: (context, index) {
                        final host = filtered[index];
                        return Dismissible(
                          key: ValueKey(host.id),
                          direction: DismissDirection.endToStart,
                          background: Container(
                            color: Colors.red,
                            alignment: Alignment.centerRight,
                            padding: const EdgeInsets.only(right: 20),
                            child: const Icon(Icons.delete, color: Colors.white),
                          ),
                          confirmDismiss: (_) async {
                            return await showDialog<bool>(
                              context: context,
                              builder: (ctx) => AlertDialog(
                                title: const Text('Delete Host'),
                                content: Text('Are you sure you want to delete "${host.name}"?'),
                                actions: [
                                  TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('Cancel')),
                                  TextButton(
                                    onPressed: () => Navigator.pop(ctx, true),
                                    child: const Text('Delete', style: TextStyle(color: Colors.red)),
                                  ),
                                ],
                              ),
                            );
                          },
                          onDismissed: (_) {
                            context.read<HostBloc>().add(HostDeleteRequested(host.id));
                          },
                          child: Padding(
                            padding: const EdgeInsets.symmetric(horizontal: 12.0, vertical: 4.0),
                            child: Card(
                              elevation: 2,
                              child: ListTile(
                                leading: CircleAvatar(
                                  backgroundColor: _statusColor(host.status).withOpacity(0.15),
                                  child: Text(
                                    host.name.isNotEmpty ? host.name[0].toUpperCase() : '?',
                                    style: TextStyle(color: _statusColor(host.status), fontWeight: FontWeight.bold),
                                  ),
                                ),
                                title: Text(host.name, style: const TextStyle(fontWeight: FontWeight.w600)),
                                subtitle: Column(
                                  crossAxisAlignment: CrossAxisAlignment.start,
                                  children: [
                                    Text('${host.address}:${host.port}'),
                                    if (host.tags.isNotEmpty)
                                      Wrap(
                                        spacing: 4,
                                        children: host.tags
                                            .take(3)
                                            .map((tag) => Chip(
                                                  label: Text(tag, style: const TextStyle(fontSize: 10)),
                                                  padding: EdgeInsets.zero,
                                                  materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
                                                  visualDensity: VisualDensity.compact,
                                                ))
                                            .toList(),
                                      ),
                                  ],
                                ),
                                isThreeLine: host.tags.isNotEmpty,
                                trailing: _StatusDot(status: host.status),
                                onTap: () => Navigator.of(context).push(
                                  MaterialPageRoute(builder: (_) => HostDetailScreen(hostId: host.id)),
                                ),
                              ),
                            ),
                          ),
                        );
                      },
                    ),
                  );
                },
              ),
            ),
          ],
        ),
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => _navigateToCreate(context),
        icon: const Icon(Icons.add),
        label: const Text('Add Host'),
      ),
    );
  }

  void _navigateToCreate(BuildContext context) {
    Navigator.of(context).push(
      MaterialPageRoute(builder: (_) => const HostCreateScreen()),
    );
  }

  void _showFilterSheet() {
    showModalBottomSheet(
      context: context,
      builder: (context) {
        return SafeArea(
          child: Padding(
            padding: const EdgeInsets.all(16.0),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('Filter by Status', style: Theme.of(context).textTheme.titleMedium),
                const SizedBox(height: 8),
                Wrap(
                  spacing: 8,
                  children: [
                    FilterChip(
                      label: const Text('Online'),
                      selected: _selectedStatusFilter == 'online',
                      onSelected: (selected) => setState(() => _selectedStatusFilter = selected ? 'online' : null),
                    ),
                    FilterChip(
                      label: const Text('Offline'),
                      selected: _selectedStatusFilter == 'offline',
                      onSelected: (selected) => setState(() => _selectedStatusFilter = selected ? 'offline' : null),
                    ),
                    FilterChip(
                      label: const Text('Connecting'),
                      selected: _selectedStatusFilter == 'connecting',
                      onSelected: (selected) => setState(() => _selectedStatusFilter = selected ? 'connecting' : null),
                    ),
                  ],
                ),
                const SizedBox(height: 16),
                Text('Filter by Organization', style: Theme.of(context).textTheme.titleMedium),
                const SizedBox(height: 8),
                Wrap(
                  spacing: 8,
                  children: [
                    FilterChip(
                      label: const Text('Personal'),
                      selected: _selectedOrgFilter == 'personal',
                      onSelected: (selected) => setState(() => _selectedOrgFilter = selected ? 'personal' : null),
                    ),
                    FilterChip(
                      label: const Text('Work'),
                      selected: _selectedOrgFilter == 'work',
                      onSelected: (selected) => setState(() => _selectedOrgFilter = selected ? 'work' : null),
                    ),
                  ],
                ),
              ],
            ),
          ),
        );
      },
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

class _StatusDot extends StatelessWidget {
  final String status;
  const _StatusDot({required this.status});

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
      width: 12,
      height: 12,
      decoration: BoxDecoration(color: color, shape: BoxShape.circle),
    );
  }
}
