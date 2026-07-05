import 'package:flutter/material.dart';

class ConnectionStatus extends StatelessWidget {
  final bool connected;

  const ConnectionStatus({super.key, required this.connected});

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(
          Icons.circle,
          size: 12,
          color: connected ? Colors.green : Colors.red,
        ),
        const SizedBox(width: 4),
        Text(connected ? 'Connected' : 'Disconnected'),
      ],
    );
  }
}
