import 'package:flutter/material.dart';

class LimitedBoxScale extends StatelessWidget {
  const LimitedBoxScale({super.key});

  @override
  Widget build(BuildContext context) {
    return LimitedBox(
      maxWidth: 100,
      maxHeight: 100,
      child: Container(color: Colors.blue, child: const Center(child: Text('Limited'))),
    );
  }
}
