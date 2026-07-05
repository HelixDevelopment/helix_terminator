import 'package:flutter/material.dart';

class PositionedScale extends StatelessWidget {
  const PositionedScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Stack(
      children: [
        Container(width: 200, height: 200, color: Colors.blue),
        Positioned.fill(child: Container(color: Colors.red.withOpacity(0.3))),
      ],
    );
  }
}
