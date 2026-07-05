import 'package:flutter/material.dart';

class ConnectionStringChip extends StatelessWidget {
  final String protocol;
  final String address;

  const ConnectionStringChip({super.key, required this.protocol, required this.address});

  @override
  Widget build(BuildContext context) {
    return Chip(
      avatar: Icon(protocol == 'SSH' ? Icons.terminal : Icons.web),
      label: Text('$protocol://$address'),
    );
  }
}
