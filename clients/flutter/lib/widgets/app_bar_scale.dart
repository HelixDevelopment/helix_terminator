import 'package:flutter/material.dart';

class AppBarScale extends StatelessWidget {
  const AppBarScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        AppBar(title: const Text('Center')),
        AppBar(leading: const Icon(Icons.menu), title: const Text('Leading')),
        AppBar(title: const Text('Actions'), actions: const [Icon(Icons.search)]),
      ],
    );
  }
}
