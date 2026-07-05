import 'package:flutter/material.dart';

class FilterChips extends StatelessWidget {
  final List<String> filters;
  final String selected;
  final ValueChanged<String>? onSelected;

  const FilterChips({super.key, required this.filters, required this.selected, this.onSelected});

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 8,
      children: filters.map((f) {
        return ChoiceChip(
          label: Text(f),
          selected: f == selected,
          onSelected: (_) => onSelected?.call(f),
        );
      }).toList(),
    );
  }
}
