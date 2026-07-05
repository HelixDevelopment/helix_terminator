import 'package:flutter/material.dart';

class ActivityFeed extends StatelessWidget {
  final List<dynamic> activities;

  const ActivityFeed({super.key, required this.activities});

  @override
  Widget build(BuildContext context) {
    return ListView.builder(
      itemCount: activities.length,
      itemBuilder: (context, index) {
        final a = activities[index];
        return ListTile(
          leading: const Icon(Icons.history),
          title: Text(a.title),
          subtitle: Text(a.timestamp),
        );
      },
    );
  }
}
