import 'package:flutter/material.dart';

class AuditLogEntry extends StatelessWidget {
  final String actor;
  final String action;
  final String target;
  final String timestamp;

  const AuditLogEntry({
    super.key,
    required this.actor,
    required this.action,
    required this.target,
    required this.timestamp,
  });

  @override
  Widget build(BuildContext context) {
    return ListTile(
      leading: const Icon(Icons.security),
      title: Text('$actor $action $target'),
      subtitle: Text(timestamp),
    );
  }
}
