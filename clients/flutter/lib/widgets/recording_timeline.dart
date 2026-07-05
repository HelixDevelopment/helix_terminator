import 'package:flutter/material.dart';

class RecordingTimeline extends StatelessWidget {
  final Duration duration;
  final Duration position;
  final ValueChanged<Duration>? onSeek;

  const RecordingTimeline({super.key, required this.duration, required this.position, this.onSeek});

  @override
  Widget build(BuildContext context) {
    final progress = duration.inSeconds > 0 ? position.inSeconds / duration.inSeconds : 0.0;
    return Slider(
      value: progress.clamp(0.0, 1.0),
      onChanged: (v) => onSeek?.call(Duration(seconds: (v * duration.inSeconds).round())),
    );
  }
}
