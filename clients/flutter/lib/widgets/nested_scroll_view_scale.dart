import 'package:flutter/material.dart';

class NestedScrollViewScale extends StatelessWidget {
  const NestedScrollViewScale({super.key});

  @override
  Widget build(BuildContext context) {
    return NestedScrollView(
      headerSliverBuilder: (context, innerBoxIsScrolled) {
        return [SliverAppBar(title: const Text('Nested'), pinned: true)];
      },
      body: ListView.builder(itemCount: 20, itemBuilder: (context, i) => ListTile(title: Text('Item $i'))),
    );
  }
}
