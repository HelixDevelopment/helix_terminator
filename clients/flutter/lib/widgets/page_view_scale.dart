import 'package:flutter/material.dart';

class PageViewScale extends StatelessWidget {
  const PageViewScale({super.key});

  @override
  Widget build(BuildContext context) {
    return PageView(
      children: [
        Container(color: Colors.red, child: const Center(child: Text('Page 1'))),
        Container(color: Colors.green, child: const Center(child: Text('Page 2'))),
        Container(color: Colors.blue, child: const Center(child: Text('Page 3'))),
      ],
    );
  }
}
