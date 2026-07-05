import 'package:flutter/material.dart';

class ContextMenu extends StatelessWidget {
  final List<PopupMenuEntry<String>> items;
  final Widget child;
  final ValueChanged<String>? onSelected;

  const ContextMenu({super.key, required this.items, required this.child, this.onSelected});

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onSecondaryTap: () {
        showMenu(
          context: context,
          position: const RelativeRect.fromLTRB(100, 100, 0, 0),
          items: items,
        );
      },
      child: child,
    );
  }
}
