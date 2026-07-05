import 'package:flutter/material.dart';

class KeyboardShortcutsOverlay extends StatelessWidget {
  final Map<String, String> shortcuts;
  final VoidCallback? onDismiss;

  const KeyboardShortcutsOverlay({super.key, required this.shortcuts, this.onDismiss});

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('Keyboard Shortcuts'),
      content: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: shortcuts.entries.map((e) => ListTile(
            leading: Chip(label: Text(e.key)),
            title: Text(e.value),
          )).toList(),
        ),
      ),
      actions: [
        TextButton(onPressed: onDismiss, child: const Text('Close')),
      ],
    );
  }
}
