import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import '../bloc/audit_bloc.dart';
import '../widgets/empty_state.dart';
import '../widgets/loading_indicator.dart';
import '../widgets/error_widget.dart' as helix_error;

class AuditLogScreen extends StatefulWidget {
  const AuditLogScreen({super.key});

  @override
  State<AuditLogScreen> createState() => _AuditLogScreenState();
}

class _AuditLogScreenState extends State<AuditLogScreen> {
  final TextEditingController _searchController = TextEditingController();
  DateTimeRange? _dateRange;
  String? _selectedAction;

  @override
  void initState() {
    super.initState();
    context.read<AuditBloc>().add(AuditLogListRequested());
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  Future<void> _pickDateRange() async {
    final picked = await showDateRangePicker(
      context: context,
      firstDate: DateTime(2020),
      lastDate: DateTime.now(),
      initialDateRange: _dateRange,
    );
    if (picked != null) {
      setState(() => _dateRange = picked);
      context.read<AuditBloc>().add(AuditLogFilterChanged(
        from: picked.start,
        to: picked.end,
        action: _selectedAction,
      ));
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Audit Log'),
        actions: [
          IconButton(
            icon: const Icon(Icons.download),
            tooltip: 'Export',
            onPressed: () {
              context.read<AuditBloc>().add(AuditLogExportRequested(
                from: _dateRange?.start,
                to: _dateRange?.end,
                action: _selectedAction,
              ));
            },
          ),
        ],
      ),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(16.0),
            child: TextField(
              controller: _searchController,
              decoration: InputDecoration(
                hintText: 'Search audit logs...',
                prefixIcon: const Icon(Icons.search),
                suffixIcon: _searchController.text.isNotEmpty
                    ? IconButton(
                        icon: const Icon(Icons.clear),
                        onPressed: () {
                          _searchController.clear();
                          context.read<AuditBloc>().add(AuditLogSearchChanged(''));
                        },
                      )
                    : null,
              ),
              onChanged: (value) {
                context.read<AuditBloc>().add(AuditLogSearchChanged(value));
              },
            ),
          ),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16.0),
            child: Row(
              children: [
                Expanded(
                  child: DropdownButtonFormField<String?>(
                    value: _selectedAction,
                    decoration: const InputDecoration(labelText: 'Action'),
                    items: [
                      const DropdownMenuItem(value: null, child: Text('All Actions')),
                      ...['login', 'logout', 'create', 'update', 'delete', 'share']
                          .map((a) => DropdownMenuItem(value: a, child: Text(a))),
                    ],
                    onChanged: (value) {
                      setState(() => _selectedAction = value);
                      context.read<AuditBloc>().add(AuditLogFilterChanged(
                        action: value,
                        from: _dateRange?.start,
                        to: _dateRange?.end,
                      ));
                    },
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: OutlinedButton.icon(
                    onPressed: _pickDateRange,
                    icon: const Icon(Icons.date_range),
                    label: Text(_dateRange == null
                        ? 'Date Range'
                        : '${_dateRange!.start.day}/${_dateRange!.start.month} - ${_dateRange!.end.day}/${_dateRange!.end.month}'),
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 8),
          Expanded(
            child: BlocConsumer<AuditBloc, AuditState>(
              listener: (context, state) {
                if (state is AuditActionSuccess) {
                  ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(state.message)));
                }
              },
              builder: (context, state) {
                if (state is AuditLoading) {
                  return const LoadingIndicator();
                }
                if (state is AuditError) {
                  return helix_error.ErrorWidget(
                    message: state.message,
                    onRetry: () => context.read<AuditBloc>().add(AuditLogListRequested()),
                  );
                }
                if (state is AuditListLoaded) {
                  final logs = state.logs.where((log) {
                    if (state.searchQuery.isEmpty) return true;
                    final q = state.searchQuery.toLowerCase();
                    return log.actor.toLowerCase().contains(q) ||
                        log.action.toLowerCase().contains(q) ||
                        log.target.toLowerCase().contains(q);
                  }).toList();

                  if (logs.isEmpty) {
                    return const EmptyState(message: 'No audit logs found');
                  }

                  return ListView.builder(
                    itemCount: logs.length,
                    itemBuilder: (context, index) {
                      final log = logs[index];
                      return Card(
                        margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
                        child: ListTile(
                          leading: CircleAvatar(
                            backgroundColor: _actionColor(log.action),
                            child: Icon(_actionIcon(log.action), color: Colors.white, size: 18),
                          ),
                          title: Text('${log.actor} ${log.action} ${log.target}'),
                          subtitle: Text(
                            '${log.timestamp.day}/${log.timestamp.month}/${log.timestamp.year} ${log.timestamp.hour}:${log.timestamp.minute.toString().padLeft(2, '0')}',
                          ),
                          trailing: Chip(
                            label: Text(log.action),
                            backgroundColor: _actionColor(log.action).withOpacity(0.2),
                            side: BorderSide(color: _actionColor(log.action)),
                          ),
                        ),
                      );
                    },
                  );
                }
                return const SizedBox.shrink();
              },
            ),
          ),
        ],
      ),
    );
  }

  Color _actionColor(String action) {
    return switch (action.toLowerCase()) {
      'login' || 'logout' => Colors.blue,
      'create' => Colors.green,
      'update' => Colors.orange,
      'delete' => Colors.red,
      'share' => Colors.purple,
      _ => Colors.grey,
    };
  }

  IconData _actionIcon(String action) {
    return switch (action.toLowerCase()) {
      'login' => Icons.login,
      'logout' => Icons.logout,
      'create' => Icons.add,
      'update' => Icons.edit,
      'delete' => Icons.delete,
      'share' => Icons.share,
      _ => Icons.info,
    };
  }
}
