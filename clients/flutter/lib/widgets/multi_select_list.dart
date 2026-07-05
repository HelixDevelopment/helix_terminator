import 'package:flutter/material.dart';

class MultiSelectList extends StatelessWidget {
  final List<String> items;
  final List<String> selected;
  final ValueChanged<List<String>>? onSelectionChanged;

  const MultiSelectList({super.key, required this.items, required this.selected, this.onSelectionChanged});

  @override
  Widget build(BuildContext context) {
    return ListView.builder(
      itemCount: items.length,
      itemBuilder: (context, index) {
        final item = items[index];
        final isSelected = selected.contains(item);
        return CheckboxListTile(
          title: Text(item),
          value: isSelected,
          onChanged: (v) {
            final newSelection = List<String>.from(selected);
            if (v == true) {
              newSelection.add(item);
            } else {
              newSelection.remove(item);
            }
            onSelectionChanged?.call(newSelection);
          },
        );
      },
    );
  }
}
