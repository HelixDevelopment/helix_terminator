import 'package:flutter/material.dart';

class AspectRatioScale extends StatelessWidget {
  const AspectRatioScale({super.key});

  @override
  Widget build(BuildContext context) {
    return AspectRatio(
      aspectRatio: 16 / 9,
      child: Container(color: Colors.blue, child: const Center(child: Text('16:9'))),
    );
  }
}
