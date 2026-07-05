import 'package:flutter/material.dart';

class SessionInfoBar extends StatelessWidget {
  final String user;
  final String host;
  final Duration duration;

  const SessionInfoBar({super.key, required this.user, required this.host, required this.duration});

  @override
  Widget build(BuildContext context) {
    final minutes = duration.inMinutes.toString().padLeft(2, '0');
    final seconds = (duration.inSeconds % 60).toString().padLeft(2, '0');
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
      color: Theme.of(context).colorScheme.surfaceContainerHighest,
      child: Row(
        children: [
          Text('$user@$host'),
          const Spacer(),
          Text('$minutes:$seconds'),
        ],
      ),
    );
  }
}
