import 'package:flutter/material.dart';

class OrgSwitcher extends StatelessWidget {
  final List<String> orgs;
  final String selected;
  final ValueChanged<String>? onChanged;

  const OrgSwitcher({super.key, required this.orgs, required this.selected, this.onChanged});

  @override
  Widget build(BuildContext context) {
    return DropdownButton<String>(
      value: selected,
      items: orgs.map((o) => DropdownMenuItem(value: o, child: Text(o))).toList(),
      onChanged: (v) => onChanged?.call(v!),
    );
  }
}
