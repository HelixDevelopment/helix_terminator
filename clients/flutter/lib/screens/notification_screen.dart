import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import '../bloc/notification_bloc.dart';
import '../widgets/empty_state.dart';
import '../widgets/loading_indicator.dart';
import '../widgets/error_widget.dart' as helix_error;

class NotificationScreen extends StatefulWidget {
  const NotificationScreen({super.key});

  @override
  State<NotificationScreen> createState() => _NotificationScreenState();
}

class _NotificationScreenState extends State<NotificationScreen> {
  final TextEditingController _searchController = TextEditingController();

  @override
  void initState() {
    super.initState();
    context.read<NotificationBloc>().add(NotificationListRequested());
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Notifications'),
        actions: [
          IconButton(
            icon: const Icon(Icons.done_all),
            tooltip: 'Mark all as read',
            onPressed: () {
              context.read<NotificationBloc>().add(NotificationMarkAllAsRead());
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
                hintText: 'Search notifications...',
                prefixIcon: const Icon(Icons.search),
                suffixIcon: _searchController.text.isNotEmpty
                    ? IconButton(
                        icon: const Icon(Icons.clear),
                        onPressed: () {
                          _searchController.clear();
                          context.read<NotificationBloc>().add(NotificationSearchChanged(''));
                        },
                      )
                    : null,
              ),
              onChanged: (value) {
                context.read<NotificationBloc>().add(NotificationSearchChanged(value));
              },
            ),
          ),
          SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            padding: const EdgeInsets.symmetric(horizontal: 16.0),
            child: Row(
              children: [
                FilterChip(
                  label: const Text('All'),
                  selected: context.select<NotificationBloc, bool>(
                    (bloc) => bloc.state is NotificationLoaded && (bloc.state as NotificationLoaded).unreadOnly == null,
                  ),
                  onSelected: (_) => context.read<NotificationBloc>().add(
                    NotificationFilterChanged(unreadOnly: null),
                  ),
                ),
                const SizedBox(width: 8),
                FilterChip(
                  label: const Text('Unread'),
                  selected: context.select<NotificationBloc, bool>(
                    (bloc) => bloc.state is NotificationLoaded && (bloc.state as NotificationLoaded).unreadOnly == true,
                  ),
                  onSelected: (_) => context.read<NotificationBloc>().add(
                    NotificationFilterChanged(unreadOnly: true),
                  ),
                ),
                const SizedBox(width: 8),
                FilterChip(
                  label: const Text('Info'),
                  selected: context.select<NotificationBloc, bool>(
                    (bloc) => bloc.state is NotificationLoaded && (bloc.state as NotificationLoaded).filterType == 'info',
                  ),
                  onSelected: (_) => context.read<NotificationBloc>().add(
                    NotificationFilterChanged(type: 'info'),
                  ),
                ),
                const SizedBox(width: 8),
                FilterChip(
                  label: const Text('Warning'),
                  selected: context.select<NotificationBloc, bool>(
                    (bloc) => bloc.state is NotificationLoaded && (bloc.state as NotificationLoaded).filterType == 'warning',
                  ),
                  onSelected: (_) => context.read<NotificationBloc>().add(
                    NotificationFilterChanged(type: 'warning'),
                  ),
                ),
                const SizedBox(width: 8),
                FilterChip(
                  label: const Text('Error'),
                  selected: context.select<NotificationBloc, bool>(
                    (bloc) => bloc.state is NotificationLoaded && (bloc.state as NotificationLoaded).filterType == 'error',
                  ),
                  onSelected: (_) => context.read<NotificationBloc>().add(
                    NotificationFilterChanged(type: 'error'),
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 8),
          Expanded(
            child: BlocConsumer<NotificationBloc, NotificationState>(
              listener: (context, state) {
                if (state is NotificationActionSuccess) {
                  ScaffoldMessenger.of(context).showSnackBar(
                    SnackBar(content: Text(state.message)),
                  );
                }
              },
              builder: (context, state) {
                if (state is NotificationLoading) {
                  return const LoadingIndicator();
                }
                if (state is NotificationError) {
                  return helix_error.ErrorWidget(
                    message: state.message,
                    onRetry: () => context.read<NotificationBloc>().add(NotificationListRequested()),
                  );
                }
                if (state is NotificationLoaded) {
                  final filtered = state.notifications.where((n) {
                    if (state.searchQuery.isEmpty) return true;
                    final q = state.searchQuery.toLowerCase();
                    return n.title.toLowerCase().contains(q) || n.body.toLowerCase().contains(q);
                  }).toList();

                  if (filtered.isEmpty) {
                    return const EmptyState(message: 'No notifications found');
                  }

                  return ListView.builder(
                    itemCount: filtered.length,
                    itemBuilder: (context, index) {
                      final notification = filtered[index];
                      return Dismissible(
                        key: ValueKey(notification.id),
                        direction: DismissDirection.endToStart,
                        background: Container(
                          color: Colors.red,
                          alignment: Alignment.centerRight,
                          padding: const EdgeInsets.only(right: 16),
                          child: const Icon(Icons.delete, color: Colors.white),
                        ),
                        onDismissed: (_) {
                          context.read<NotificationBloc>().add(NotificationDelete(notification.id));
                        },
                        child: ListTile(
                          leading: CircleAvatar(
                            backgroundColor: notification.read
                                ? Colors.grey.shade300
                                : Theme.of(context).colorScheme.primaryContainer,
                            child: Icon(
                              notification.read ? Icons.notifications_none : Icons.notifications,
                              color: notification.read
                                  ? Colors.grey
                                  : Theme.of(context).colorScheme.primary,
                            ),
                          ),
                          title: Text(
                            notification.title,
                            style: TextStyle(
                              fontWeight: notification.read ? FontWeight.normal : FontWeight.bold,
                            ),
                          ),
                          subtitle: Text(notification.body),
                          trailing: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              if (!notification.read)
                                IconButton(
                                  icon: const Icon(Icons.done),
                                  tooltip: 'Mark as read',
                                  onPressed: () {
                                    context.read<NotificationBloc>().add(NotificationMarkAsRead(notification.id));
                                  },
                                ),
                              IconButton(
                                icon: const Icon(Icons.delete_outline),
                                tooltip: 'Delete',
                                onPressed: () {
                                  context.read<NotificationBloc>().add(NotificationDelete(notification.id));
                                },
                              ),
                            ],
                          ),
                          onTap: () {
                            if (!notification.read) {
                              context.read<NotificationBloc>().add(NotificationMarkAsRead(notification.id));
                            }
                          },
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
}
