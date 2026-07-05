import 'package:flutter/material.dart';

class DragTargetScale extends StatelessWidget {
  const DragTargetScale({super.key});

  @override
  Widget build(BuildContext context) {
    return DragTarget<String>(
      onAccept: (data) {},
      builder: (context, candidateData, rejectedData) {
        return Container(
          width: 100,
          height: 100,
          color: candidateData.isNotEmpty ? Colors.green : Colors.red,
          child: const Center(child: Text('Drop here')),
        );
      },
    );
  }
}
