import 'package:flutter/material.dart';

class RecordingCard extends StatelessWidget {
  final String title;
  final String duration;
  final VoidCallback? onTap;

  const RecordingCard({super.key, required this.title, required this.duration, this.onTap});

  @override
  Widget build(BuildContext context) {
    return Card(
      child: ListTile(
        leading: const Icon(Icons.videocam),
        title: Text(title),
        subtitle: Text(duration),
        onTap: onTap,
      ),
    );
  }
}
