import 'package:flutter/material.dart';

class SlideIn extends StatelessWidget {
  final Widget child;
  final Axis direction;

  const SlideIn({super.key, required this.child, this.direction = Axis.vertical});

  @override
  Widget build(BuildContext context) {
    // TODO: implement animated slide-in
    return child;
  }
}
