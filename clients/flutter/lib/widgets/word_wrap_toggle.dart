import 'package:flutter/material.dart';

class WordWrapToggle extends StatelessWidget {
  final bool enabled;
  final ValueChanged<bool>? onChanged;

  const WordWrapToggle({super.key, required this.enabled, this.onChanged});

  @override
  Widget build(BuildContext context) {
    return SwitchListTile(
      title: const Text('Word Wrap'),
      value: enabled,
      onChanged: onChanged,
    );
  }
}
