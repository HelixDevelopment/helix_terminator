import 'package:flutter/material.dart';

class LineNumbersToggle extends StatelessWidget {
  final bool enabled;
  final ValueChanged<bool>? onChanged;

  const LineNumbersToggle({super.key, required this.enabled, this.onChanged});

  @override
  Widget build(BuildContext context) {
    return SwitchListTile(
      title: const Text('Line Numbers'),
      value: enabled,
      onChanged: onChanged,
    );
  }
}
