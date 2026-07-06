import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import '../bloc/recording_bloc.dart';
import '../widgets/loading_indicator.dart';
import '../widgets/error_widget.dart' as helix_error;

class RecordingPlayerScreen extends StatefulWidget {
  final String recordingId;
  const RecordingPlayerScreen({super.key, required this.recordingId});

  @override
  State<RecordingPlayerScreen> createState() => _RecordingPlayerScreenState();
}

class _RecordingPlayerScreenState extends State<RecordingPlayerScreen> {
  bool _isPlaying = false;
  double _progress = 0.0;
  double _speed = 1.0;

  @override
  void initState() {
    super.initState();
    context.read<RecordingBloc>().add(RecordingLoadRequested(widget.recordingId));
  }

  String _formatDuration(Duration duration) {
    final minutes = duration.inMinutes.remainder(60).toString().padLeft(2, '0');
    final seconds = duration.inSeconds.remainder(60).toString().padLeft(2, '0');
    return '$minutes:$seconds';
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Recording Player'),
      ),
      body: BlocBuilder<RecordingBloc, RecordingState>(
        builder: (context, state) {
          if (state is RecordingLoading) {
            return const LoadingIndicator();
          }
          if (state is RecordingError) {
            return helix_error.ErrorWidget(
              message: state.message,
              onRetry: () => context.read<RecordingBloc>().add(RecordingLoadRequested(widget.recordingId)),
            );
          }
          if (state is RecordingDetailLoaded) {
            final recording = state.recording;
            return Column(
              children: [
                Expanded(
                  child: Container(
                    color: Colors.black,
                    child: Center(
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Icon(
                            _isPlaying ? Icons.pause_circle : Icons.play_circle,
                            size: 80,
                            color: Colors.white,
                          ),
                          const SizedBox(height: 16),
                          Text(
                            recording.title,
                            style: const TextStyle(color: Colors.white, fontSize: 18),
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
                Container(
                  padding: const EdgeInsets.all(16),
                  color: Theme.of(context).colorScheme.surface,
                  child: Column(
                    children: [
                      Row(
                        mainAxisAlignment: MainAxisAlignment.spaceBetween,
                        children: [
                          Text(_formatDuration(Duration(seconds: (_progress * recording.duration.inSeconds).toInt()))),
                          Text(_formatDuration(recording.duration)),
                        ],
                      ),
                      Slider(
                        value: _progress,
                        onChanged: (value) {
                          setState(() => _progress = value);
                        },
                      ),
                      Row(
                        mainAxisAlignment: MainAxisAlignment.center,
                        children: [
                          IconButton(
                            icon: Icon(_isPlaying ? Icons.pause : Icons.play_arrow),
                            iconSize: 48,
                            onPressed: () {
                              setState(() => _isPlaying = !_isPlaying);
                            },
                          ),
                          const SizedBox(width: 16),
                          PopupMenuButton<double>(
                            initialValue: _speed,
                            onSelected: (value) => setState(() => _speed = value),
                            itemBuilder: (context) => [
                              const PopupMenuItem(value: 0.5, child: Text('0.5x')),
                              const PopupMenuItem(value: 1.0, child: Text('1.0x')),
                              const PopupMenuItem(value: 1.5, child: Text('1.5x')),
                              const PopupMenuItem(value: 2.0, child: Text('2.0x')),
                            ],
                            child: Chip(label: Text('${_speed}x')),
                          ),
                        ],
                      ),
                    ],
                  ),
                ),
              ],
            );
          }
          return const SizedBox.shrink();
        },
      ),
    );
  }
}
