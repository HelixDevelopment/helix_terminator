import 'package:flutter/material.dart';

class EasingCurvePreview extends StatelessWidget {
  const EasingCurvePreview({super.key});

  @override
  Widget build(BuildContext context) {
    final curves = [
      Curves.linear,
      Curves.easeIn,
      Curves.easeOut,
      Curves.easeInOut,
      Curves.bounceIn,
      Curves.bounceOut,
      Curves.elasticIn,
      Curves.elasticOut,
    ];
    return Column(
      children: curves.map((c) {
        return ListTile(
          title: Text(c.toString()),
          trailing: const Icon(Icons.animation),
        );
      }).toList(),
    );
  }
}
