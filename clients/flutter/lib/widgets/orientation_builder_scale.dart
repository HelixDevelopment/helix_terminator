import 'package:flutter/material.dart';

class OrientationBuilderScale extends StatelessWidget {
  const OrientationBuilderScale({super.key});

  @override
  Widget build(BuildContext context) {
    return OrientationBuilder(
      builder: (context, orientation) {
        return Container(
          color: orientation == Orientation.portrait ? Colors.blue : Colors.green,
          child: const Center(child: Text('Orientation')),
        );
      },
    );
  }
}
