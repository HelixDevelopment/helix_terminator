import 'package:flutter/material.dart';

class WrapScale extends StatelessWidget {
  const WrapScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: List.generate(10, (i) => Chip(label: Text('Chip $i'))),
    );
  }
}
