import 'package:flutter/material.dart';

class ContainerCard extends StatelessWidget {
  final String name;
  final String status;
  final VoidCallback? onTap;

  const ContainerCard({super.key, required this.name, required this.status, this.onTap});

  @override
  Widget build(BuildContext context) {
    return Card(
      child: ListTile(
        leading: const Icon(Icons.view_agenda),
        title: Text(name),
        subtitle: Text(status),
        trailing: StatusChip(label: status, color: status == 'running' ? Colors.green : Colors.red),
        onTap: onTap,
      ),
    );
  }
}

// Forward declaration to avoid import cycle; in real app use proper import
class StatusChip extends StatelessWidget {
  final String label;
  final Color color;
  const StatusChip({super.key, required this.label, required this.color});
  @override
  Widget build(BuildContext context) => Chip(label: Text(label, style: const TextStyle(color: Colors.white)), backgroundColor: color);
}
