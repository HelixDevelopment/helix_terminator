import 'package:flutter/material.dart';

class EnvironmentBadge extends StatelessWidget {
  final String environment;

  const EnvironmentBadge({super.key, required this.environment});

  @override
  Widget build(BuildContext context) {
    final color = switch (environment) {
      'production' => Colors.red,
      'staging' => Colors.orange,
      'development' => Colors.blue,
      _ => Colors.grey,
    };
    return Chip(
      label: Text(environment.toUpperCase()),
      backgroundColor: color.withOpacity(0.2),
      side: BorderSide(color: color),
    );
  }
}
