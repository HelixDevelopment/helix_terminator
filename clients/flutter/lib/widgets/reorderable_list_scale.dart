import 'package:flutter/material.dart';

class ReorderableListScale extends StatelessWidget {
  const ReorderableListScale({super.key});

  @override
  Widget build(BuildContext context) {
    return ReorderableListView(
      shrinkWrap: true,
      onReorder: (oldIndex, newIndex) {},
      children: List.generate(5, (i) => ListTile(key: ValueKey(i), title: Text('Item $i'))),
    );
  }
}
