import 'package:flutter/material.dart';

class PortForwardCard extends StatelessWidget {
  final String localPort;
  final String remotePort;
  final String host;
  final VoidCallback? onDelete;

  const PortForwardCard({
    super.key,
    required this.localPort,
    required this.remotePort,
    required this.host,
    this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      child: ListTile(
        title: Text('$localPort -> $remotePort'),
        subtitle: Text(host),
        trailing: IconButton(
          icon: const Icon(Icons.delete),
          onPressed: onDelete,
        ),
      ),
    );
  }
}
