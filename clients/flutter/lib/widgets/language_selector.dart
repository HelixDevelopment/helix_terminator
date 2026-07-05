import 'package:flutter/material.dart';

class LanguageSelector extends StatelessWidget {
  final String selected;
  final ValueChanged<String>? onChanged;

  const LanguageSelector({super.key, required this.selected, this.onChanged});

  @override
  Widget build(BuildContext context) {
    return DropdownButton<String>(
      value: selected,
      items: const [
        DropdownMenuItem(value: 'en', child: Text('English')),
        DropdownMenuItem(value: 'es', child: Text('Spanish')),
        DropdownMenuItem(value: 'fr', child: Text('French')),
        DropdownMenuItem(value: 'de', child: Text('German')),
        DropdownMenuItem(value: 'ja', child: Text('Japanese')),
      ],
      onChanged: (v) => onChanged?.call(v!),
    );
  }
}
