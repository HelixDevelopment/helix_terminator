import 'package:flutter/material.dart';

class PageViewBuilderScale extends StatelessWidget {
  const PageViewBuilderScale({super.key});

  @override
  Widget build(BuildContext context) {
    return PageView.builder(
      itemCount: 10,
      itemBuilder: (context, index) {
        return Container(
          color: Colors.primaries[index % Colors.primaries.length],
          child: Center(child: Text('Page $index')),
        );
      },
    );
  }
}
