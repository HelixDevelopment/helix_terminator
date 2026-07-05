import 'package:flutter/material.dart';

class WorkspaceCard extends StatelessWidget {
  final String name;
  final int hostCount;
  final VoidCallback? onTap;

  const WorkspaceCard({super.key, required this.name, required this.hostCount, this.onTap});

  @override
  Widget build(BuildContext context) {
    return Card(
      child: ListTile(
        leading: const Icon(Icons.workspaces),
        title: Text(name),
        subtitle: Text('$hostCount hosts'),
        onTap: onTap,
      ),
    );
  }
}
