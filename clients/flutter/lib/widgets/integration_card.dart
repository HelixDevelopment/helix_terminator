import 'package:flutter/material.dart';

class IntegrationCard extends StatelessWidget {
  final String name;
  final String description;
  final bool connected;
  final VoidCallback? onConnect;

  const IntegrationCard({
    super.key,
    required this.name,
    required this.description,
    required this.connected,
    this.onConnect,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      child: ListTile(
        leading: const Icon(Icons.extension),
        title: Text(name),
        subtitle: Text(description),
        trailing: connected
            ? const Chip(label: Text('Connected'))
            : ElevatedButton(onPressed: onConnect, child: const Text('Connect')),
      ),
    );
  }
}
