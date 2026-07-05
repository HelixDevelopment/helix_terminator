import 'package:flutter/material.dart';

class PermissionChip extends StatelessWidget {
  final String permission;
  final bool granted;

  const PermissionChip({super.key, required this.permission, required this.granted});

  @override
  Widget build(BuildContext context) {
    return Chip(
      avatar: Icon(granted ? Icons.check_circle : Icons.cancel, color: granted ? Colors.green : Colors.red),
      label: Text(permission),
    );
  }
}
