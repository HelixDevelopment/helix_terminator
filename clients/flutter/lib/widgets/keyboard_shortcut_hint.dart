import 'package:flutter/material.dart';

class KeyboardShortcutHint extends StatelessWidget {
  final String shortcut;
  final String description;

  const KeyboardShortcutHint({super.key, required this.shortcut, required this.description});

  @override
  Widget build(BuildContext context) {
    return ListTile(
      leading: Chip(label: Text(shortcut)),
      title: Text(description),
    );
  }
}
