import 'package:flutter/material.dart';

class TimelineItem extends StatelessWidget {
  final String title;
  final String subtitle;
  final bool isFirst;
  final bool isLast;

  const TimelineItem({
    super.key,
    required this.title,
    required this.subtitle,
    this.isFirst = false,
    this.isLast = false,
  });

  @override
  Widget build(BuildContext context) {
    return ListTile(
      leading: Column(
        children: [
          if (!isFirst) Container(width: 2, height: 16, color: Colors.grey),
          const Icon(Icons.circle, size: 12),
          if (!isLast) Container(width: 2, height: 16, color: Colors.grey),
        ],
      ),
      title: Text(title),
      subtitle: Text(subtitle),
    );
  }
}
