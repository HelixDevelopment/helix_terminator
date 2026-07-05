import 'package:flutter/material.dart';

class StaggeredGrid extends StatelessWidget {
  final List<Widget> children;
  final int crossAxisCount;

  const StaggeredGrid({super.key, required this.children, this.crossAxisCount = 2});

  @override
  Widget build(BuildContext context) {
    // TODO: replace with flutter_staggered_grid_view
    return GridView.count(
      crossAxisCount: crossAxisCount,
      children: children,
    );
  }
}
