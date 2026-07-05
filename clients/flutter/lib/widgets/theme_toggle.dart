import 'package:flutter/material.dart';

class ThemeToggle extends StatelessWidget {
  final bool isDark;
  final ValueChanged<bool>? onChanged;

  const ThemeToggle({super.key, required this.isDark, this.onChanged});

  @override
  Widget build(BuildContext context) {
    return SwitchListTile(
      title: const Text('Dark Mode'),
      value: isDark,
      onChanged: onChanged,
    );
  }
}
