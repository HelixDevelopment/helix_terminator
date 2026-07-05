import 'package:flutter/material.dart';

class LayoutBuilderScale extends StatelessWidget {
  const LayoutBuilderScale({super.key});

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        if (constraints.maxWidth > 600) {
          return Row(children: [Container(width: 200, height: 100, color: Colors.blue)]);
        }
        return Column(children: [Container(width: 200, height: 100, color: Colors.blue)]);
      },
    );
  }
}
