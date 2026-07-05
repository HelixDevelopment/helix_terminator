import 'package:flutter/material.dart';

class TreeViewStub extends StatelessWidget {
  final List<dynamic> nodes;
  final ValueChanged<dynamic>? onNodeTap;

  const TreeViewStub({super.key, required this.nodes, this.onNodeTap});

  @override
  Widget build(BuildContext context) {
    // TODO: integrate flutter_tree_view
    return ListView.builder(
      itemCount: nodes.length,
      itemBuilder: (context, index) => ListTile(
        title: Text(nodes[index].toString()),
        onTap: () => onNodeTap?.call(nodes[index]),
      ),
    );
  }
}
