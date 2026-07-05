import 'package:flutter/material.dart';

class ResizablePanel extends StatelessWidget {
  final Widget child;
  final double initialWidth;

  const ResizablePanel({super.key, required this.child, this.initialWidth = 300});

  @override
  Widget build(BuildContext context) {
    // TODO: implement drag-to-resize
    return SizedBox(
      width: initialWidth,
      child: child,
    );
  }
}
