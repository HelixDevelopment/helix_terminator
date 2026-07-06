import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import '../bloc/host_bloc.dart';
import '../models/host.dart';
import '../models/workspace.dart';
import '../models/snippet.dart';
import 'host_detail_screen.dart';

class SearchScreen extends StatefulWidget {
  const SearchScreen({super.key});

  @override
  State<SearchScreen> createState() => _SearchScreenState();
}

class _SearchScreenState extends State<SearchScreen> {
  final TextEditingController _searchController = TextEditingController();
  String _query = '';
  String _filterType = 'all';
  final List<String> _recentSearches = [];

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  void _addRecentSearch(String query) {
    if (query.trim().isEmpty) return;
    setState(() {
      _recentSearches.remove(query);
      _recentSearches.insert(0, query);
      if (_recentSearches.length > 10) _recentSearches.removeLast();
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(
        title: SearchBar(
          controller: _searchController,
          hintText: 'Search hosts, workspaces, snippets...',
          leading: const Icon(Icons.search),
          trailing: _query.isNotEmpty
              ? [
                  IconButton(
                    icon: const Icon(Icons.clear),
                    onPressed: () {
                      _searchController.clear();
                      setState(() => _query = '');
                    },
                  ),
                ]
              : null,
          onChanged: (value) => setState(() => _query = value),
          onSubmitted: (value) => _addRecentSearch(value),
          elevation: const WidgetStatePropertyAll(0),
        ),
        actions: [
          PopupMenuButton<String>(
            icon: const Icon(Icons.filter_list),
            onSelected: (value) => setState(() => _filterType = value),
            itemBuilder: (_) => [
              const PopupMenuItem(value: 'all', child: Text('All')),
              const PopupMenuItem(value: 'hosts', child: Text('Hosts only')),
              const PopupMenuItem(value: 'workspaces', child: Text('Workspaces only')),
              const PopupMenuItem(value: 'snippets', child: Text('Snippets only')),
            ],
          ),
        ],
      ),
      body: _query.isEmpty ? _buildRecentSearches(theme) : _buildSearchResults(theme),
    );
  }

  Widget _buildRecentSearches(ThemeData theme) {
    if (_recentSearches.isEmpty) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(Icons.search, size: 64, color: theme.colorScheme.onSurfaceVariant.withOpacity(0.4)),
            const SizedBox(height: 16),
            Text('Start typing to search', style: theme.textTheme.titleMedium),
            const SizedBox(height: 8),
            Text('Search across hosts, workspaces, and snippets', style: theme.textTheme.bodyMedium?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
          ],
        ),
      );
    }

    return ListView(
      padding: const EdgeInsets.all(16.0),
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text('Recent Searches', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
            TextButton(
              onPressed: () => setState(() => _recentSearches.clear()),
              child: const Text('Clear'),
            ),
          ],
        ),
        const SizedBox(height: 8),
        ..._recentSearches.map((search) => ListTile(
              leading: const Icon(Icons.history),
              title: Text(search),
              trailing: IconButton(
                icon: const Icon(Icons.north_west),
                onPressed: () {
                  _searchController.text = search;
                  setState(() => _query = search);
                },
              ),
              onTap: () {
                _searchController.text = search;
                setState(() => _query = search);
              },
            )),
      ],
    );
  }

  Widget _buildSearchResults(ThemeData theme) {
    return BlocBuilder<HostBloc, HostState>(
      builder: (context, state) {
        List<Host> hosts = [];
        if (state is HostLoaded) {
          hosts = state.hosts;
        } else if (state is HostLoading && state.previousHosts != null) {
          hosts = state.previousHosts!;
        } else if (state is HostError && state.previousHosts != null) {
          hosts = state.previousHosts!;
        }

        final filteredHosts = _filterType == 'all' || _filterType == 'hosts'
            ? hosts.where((h) =>
                h.name.toLowerCase().contains(_query.toLowerCase()) ||
                h.address.toLowerCase().contains(_query.toLowerCase()) ||
                h.tags.any((t) => t.toLowerCase().contains(_query.toLowerCase())))
            : <Host>[];

        final workspaces = _filterType == 'all' || _filterType == 'workspaces'
            ? _mockWorkspaces().where((w) =>
                w.name.toLowerCase().contains(_query.toLowerCase()) ||
                (w.description?.toLowerCase().contains(_query.toLowerCase()) ?? false))
            : <Workspace>[];

        final snippets = _filterType == 'all' || _filterType == 'snippets'
            ? _mockSnippets().where((s) =>
                s.title.toLowerCase().contains(_query.toLowerCase()) ||
                s.content.toLowerCase().contains(_query.toLowerCase()) ||
                s.language.toLowerCase().contains(_query.toLowerCase()))
            : <Snippet>[];

        final hasResults = filteredHosts.isNotEmpty || workspaces.isNotEmpty || snippets.isNotEmpty;

        if (!hasResults) {
          return Center(
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Icon(Icons.search_off, size: 64, color: theme.colorScheme.onSurfaceVariant.withOpacity(0.4)),
                const SizedBox(height: 16),
                Text('No results for "$_query"', style: theme.textTheme.titleMedium),
              ],
            ),
          );
        }

        return ListView(
          padding: const EdgeInsets.all(16.0),
          children: [
            if (filteredHosts.isNotEmpty) ...[
              _SectionHeader(title: 'Hosts (${filteredHosts.length})', icon: Icons.dns),
              ...filteredHosts.map((host) => Card(
                    elevation: 2,
                    child: ListTile(
                      leading: CircleAvatar(
                        backgroundColor: _statusColor(host.status).withOpacity(0.15),
                        child: Text(
                          host.name.isNotEmpty ? host.name[0].toUpperCase() : '?',
                          style: TextStyle(color: _statusColor(host.status), fontWeight: FontWeight.bold),
                        ),
                      ),
                      title: Text(host.name),
                      subtitle: Text('${host.address}:${host.port}'),
                      trailing: _StatusDot(status: host.status),
                      onTap: () => Navigator.of(context).push(
                        MaterialPageRoute(builder: (_) => HostDetailScreen(hostId: host.id)),
                      ),
                    ),
                  )),
              const SizedBox(height: 16),
            ],
            if (workspaces.isNotEmpty) ...[
              _SectionHeader(title: 'Workspaces (${workspaces.length})', icon: Icons.workspaces),
              ...workspaces.map((workspace) => Card(
                    elevation: 2,
                    child: ListTile(
                      leading: const CircleAvatar(child: Icon(Icons.workspaces)),
                      title: Text(workspace.name),
                      subtitle: Text(workspace.description ?? 'No description'),
                      onTap: () {},
                    ),
                  )),
              const SizedBox(height: 16),
            ],
            if (snippets.isNotEmpty) ...[
              _SectionHeader(title: 'Snippets (${snippets.length})', icon: Icons.code),
              ...snippets.map((snippet) => Card(
                    elevation: 2,
                    child: ListTile(
                      leading: const CircleAvatar(child: Icon(Icons.code)),
                      title: Text(snippet.title),
                      subtitle: Text('${snippet.language} · ${snippet.content.length} chars'),
                      onTap: () {},
                    ),
                  )),
            ],
          ],
        );
      },
    );
  }

  List<Workspace> _mockWorkspaces() {
    return [
      Workspace(id: '1', name: 'Production Infrastructure', description: 'Main production servers', createdAt: DateTime.now()),
      Workspace(id: '2', name: 'Development Environment', description: 'Dev and staging hosts', createdAt: DateTime.now()),
      Workspace(id: '3', name: 'Personal Projects', createdAt: DateTime.now()),
    ];
  }

  List<Snippet> _mockSnippets() {
    return [
      Snippet(id: '1', title: 'Docker Compose Template', content: 'version: "3"\nservices:', language: 'yaml', createdAt: DateTime.now()),
      Snippet(id: '2', title: 'Nginx Reverse Proxy', content: 'server { listen 80; }', language: 'nginx', createdAt: DateTime.now()),
      Snippet(id: '3', title: 'Systemd Service', content: '[Unit]\nDescription=...', language: 'ini', createdAt: DateTime.now()),
    ];
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

class _SectionHeader extends StatelessWidget {
  final String title;
  final IconData icon;

  const _SectionHeader({required this.title, required this.icon});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8.0),
      child: Row(
        children: [
          Icon(icon, size: 20, color: Theme.of(context).colorScheme.primary),
          const SizedBox(width: 8),
          Text(title, style: Theme.of(context).textTheme.titleSmall?.copyWith(fontWeight: FontWeight.bold, color: Theme.of(context).colorScheme.primary)),
        ],
      ),
    );
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
