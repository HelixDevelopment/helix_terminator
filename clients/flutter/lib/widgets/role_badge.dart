import 'package:flutter/material.dart';

class RoleBadge extends StatelessWidget {
  final String role;

  const RoleBadge({super.key, required this.role});

  @override
  Widget build(BuildContext context) {
    final color = switch (role) {
      'admin' => Colors.red,
      'editor' => Colors.orange,
      'viewer' => Colors.blue,
      _ => Colors.grey,
    };
    return Chip(
      label: Text(role),
      backgroundColor: color.withOpacity(0.2),
      side: BorderSide(color: color),
    );
  }
}
