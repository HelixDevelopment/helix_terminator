import 'package:flutter/material.dart';

class InlineActionMenu extends StatelessWidget {
  final List<PopupMenuEntry<String>> items;
  final ValueChanged<String>? onSelected;

  const InlineActionMenu({super.key, required this.items, this.onSelected});

  @override
  Widget build(BuildContext context) {
    return PopupMenuButton<String>(
      onSelected: onSelected,
      itemBuilder: (_) => items,
    );
  }
}
