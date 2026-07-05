import 'package:flutter/material.dart';

class PlatformNotification extends StatelessWidget {
  final String title;
  final String body;

  const PlatformNotification({super.key, required this.title, required this.body});

  @override
  Widget build(BuildContext context) {
    // TODO: implement flutter_local_notifications or desktop notifications
    return ListTile(
      leading: const Icon(Icons.notifications),
      title: Text(title),
      subtitle: Text(body),
    );
  }
}
