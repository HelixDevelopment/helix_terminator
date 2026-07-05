import 'package:flutter/material.dart';

class HostTag extends StatelessWidget {
  final String label;
  final Color color;

  const HostTag({super.key, required this.label, required this.color});

  @override
  Widget build(BuildContext context) {
    return Chip(
      label: Text(label, style: const TextStyle(fontSize: 10)),
      backgroundColor: color.withOpacity(0.2),
      side: BorderSide(color: color),
      padding: EdgeInsets.zero,
    );
  }
}
