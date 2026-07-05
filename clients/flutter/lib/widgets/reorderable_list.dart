import 'package:flutter/material.dart';

class ReorderableList extends StatelessWidget {
  final List<Widget> children;
  final ValueChanged<List<Widget>>? onReorder;

  const ReorderableList({super.key, required this.children, this.onReorder});

  @override
  Widget build(BuildContext context) {
    return ReorderableListView(
      onReorder: (oldIndex, newIndex) {
        // TODO: implement reorder logic
      },
      children: children,
    );
  }
}
