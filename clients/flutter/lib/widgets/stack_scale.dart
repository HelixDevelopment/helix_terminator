import 'package:flutter/material.dart';

class StackScale extends StatelessWidget {
  const StackScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Stack(
      children: [
        Container(width: 200, height: 200, color: Colors.blue),
        Positioned(top: 50, left: 50, child: Container(width: 100, height: 100, color: Colors.red)),
      ],
    );
  }
}
