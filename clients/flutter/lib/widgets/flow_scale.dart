import 'package:flutter/material.dart';

class FlowScale extends StatelessWidget {
  const FlowScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Flow(
      delegate: _FlowDelegate(),
      children: List.generate(5, (i) => Container(width: 50, height: 50, color: Colors.primaries[i])),
    );
  }
}

class _FlowDelegate extends FlowDelegate {
  @override
  void paintChildren(FlowPaintingContext context) {
    for (int i = 0; i < context.childCount; i++) {
      context.paintChild(i, transform: Matrix4.translationValues(i * 60.0, 0, 0));
    }
  }

  @override
  bool shouldRepaint(covariant FlowDelegate oldDelegate) => false;
}
