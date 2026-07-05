import 'package:flutter/material.dart';

class TerminalThemeSelector extends StatelessWidget {
  final String selectedTheme;
  final ValueChanged<String>? onChanged;

  const TerminalThemeSelector({super.key, required this.selectedTheme, this.onChanged});

  @override
  Widget build(BuildContext context) {
    return DropdownButton<String>(
      value: selectedTheme,
      items: const [
        DropdownMenuItem(value: 'dark', child: Text('Dark')),
        DropdownMenuItem(value: 'light', child: Text('Light')),
        DropdownMenuItem(value: 'solarized', child: Text('Solarized')),
        DropdownMenuItem(value: 'monokai', child: Text('Monokai')),
      ],
      onChanged: (v) => onChanged?.call(v!),
    );
  }
}
