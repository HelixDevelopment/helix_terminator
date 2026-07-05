import 'package:flutter/material.dart';

class SliverGridScale extends StatelessWidget {
  const SliverGridScale({super.key});

  @override
  Widget build(BuildContext context) {
    return CustomScrollView(
      slivers: [
        SliverGrid(
          gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(crossAxisCount: 3),
          delegate: SliverChildBuilderDelegate((_, i) => Container(color: Colors.blue), childCount: 9),
        ),
      ],
    );
  }
}
