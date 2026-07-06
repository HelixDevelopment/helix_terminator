import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import '../bloc/recording_bloc.dart';
import '../widgets/empty_state.dart';
import '../widgets/loading_indicator.dart';
import '../widgets/error_widget.dart' as helix_error;
import '../screens/recording_player_screen.dart';

class RecordingListScreen extends StatefulWidget {
  const RecordingListScreen({super.key});

  @override
  State<RecordingListScreen> createState() => _RecordingListScreenState();
}

class _RecordingListScreenState extends State<RecordingListScreen> {
  final TextEditingController _searchController = TextEditingController();

  @override
  void initState() {
    super.initState();
    context.read<RecordingBloc>().add(RecordingListRequested());
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  String _formatDuration(Duration duration) {
    final hours = duration.inHours;
    final minutes = duration.inMinutes.remainder(60);
    final seconds = duration.inSeconds.remainder(60);
    if (hours > 0) {
      return '${hours}h ${minutes}m ${seconds}s';
    }
    return '${minutes}m ${seconds}s';
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Recordings'),
      ),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(16.0),
            child: TextField(
              controller: _searchController,
              decoration: InputDecoration(
                hintText: 'Search recordings...',
                prefixIcon: const Icon(Icons.search),
                suffixIcon: _searchController.text.isNotEmpty
                    ? IconButton(
                        icon: const Icon(Icons.clear),
                        onPressed: () {
                          _searchController.clear();
                          context.read<RecordingBloc>().add(RecordingSearchChanged(''));
                        },
                      )
                    : null,
              ),
              onChanged: (value) {
                context.read<RecordingBloc>().add(RecordingSearchChanged(value));
              },
            ),
          ),
          Expanded(
            child: BlocConsumer<RecordingBloc, RecordingState>(
              listener: (context, state) {
                if (state is RecordingActionSuccess) {
                  ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(state.message)));
                }
              },
              builder: (context, state) {
                if (state is RecordingLoading) {
                  return const LoadingIndicator();
                }
                if (state is RecordingError) {
                  return helix_error.ErrorWidget(
                    message: state.message,
                    onRetry: () => context.read<RecordingBloc>().add(RecordingListRequested()),
                  );
                }
                if (state is RecordingListLoaded) {
                  final recordings = state.recordings.where((r) {
                    if (state.searchQuery.isEmpty) return true;
                    final q = state.searchQuery.toLowerCase();
                    return r.title.toLowerCase().contains(q) || r.sessionId.toLowerCase().contains(q);
                  }).toList();

                  if (recordings.isEmpty) {
                    return const EmptyState(message: 'No recordings found');
                  }

                  return ListView.builder(
                    itemCount: recordings.length,
                    itemBuilder: (context, index) {
                      final recording = recordings[index];
                      return Card(
                        margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
                        child: ListTile(
                          leading: Container(
                            width: 56,
                            height: 56,
                            decoration: BoxDecoration(
                              color: Theme.of(context).colorScheme.primaryContainer,
                              borderRadius: BorderRadius.circular(8),
                            ),
                            child: Icon(
                              Icons.videocam,
                              color: Theme.of(context).colorScheme.onPrimaryContainer,
                            ),
                          ),
                          title: Text(recording.title),
                          subtitle: Row(
                            children: [
                              const Icon(Icons.timer, size: 14),
                              const SizedBox(width: 4),
                              Text(_formatDuration(recording.duration)),
                              const SizedBox(width: 12),
                              Text(
                                '${recording.createdAt.day}/${recording.createdAt.month}/${recording.createdAt.year}',
                                style: Theme.of(context).textTheme.bodySmall,
                              ),
                            ],
                          ),
                          trailing: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              IconButton(
                                icon: const Icon(Icons.play_arrow),
                                tooltip: 'Play',
                                onPressed: () {
                                  Navigator.push(
                                    context,
                                    MaterialPageRoute(
                                      builder: (_) => RecordingPlayerScreen(recordingId: recording.id),
                                    ),
                                  );
                                },
                              ),
                              IconButton(
                                icon: const Icon(Icons.delete_outline),
                                tooltip: 'Delete',
                                onPressed: () {
                                  context.read<RecordingBloc>().add(RecordingDelete(recording.id));
                                },
                              ),
                            ],
                          ),
                          onTap: () {
                            Navigator.push(
                              context,
                              MaterialPageRoute(
                                builder: (_) => RecordingPlayerScreen(recordingId: recording.id),
                              ),
                            );
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
