import 'package:flutter/material.dart';

class GroupSelector extends StatelessWidget {
  final List<String> groups;
  final String selected;
  final ValueChanged<String>? onChanged;

  const GroupSelector({super.key, required this.groups, required this.selected, this.onChanged});

  @override
  Widget build(BuildContext context) {
    return DropdownButton<String>(
      value: selected,
      items: groups.map((g) => DropdownMenuItem(value: g, child: Text(g))).toList(),
      onChanged: (v) => onChanged?.call(v!),
    );
  }
}
