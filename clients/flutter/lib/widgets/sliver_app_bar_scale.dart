import 'package:flutter/material.dart';

class SliverAppBarScale extends StatelessWidget {
  const SliverAppBarScale({super.key});

  @override
  Widget build(BuildContext context) {
    return CustomScrollView(
      slivers: [
        const SliverAppBar(
          expandedHeight: 200,
          flexibleSpace: FlexibleSpaceBar(title: Text('Sliver AppBar')),
        ),
        SliverList(delegate: SliverChildBuilderDelegate((_, i) => ListTile(title: Text('Item $i')), childCount: 20)),
      ],
    );
  }
}
