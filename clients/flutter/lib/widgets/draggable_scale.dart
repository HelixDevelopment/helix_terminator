import 'package:flutter/material.dart';

class DraggableScale extends StatelessWidget {
  const DraggableScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Draggable<String>(
      data: 'data',
      feedback: Container(width: 100, height: 100, color: Colors.blue.withOpacity(0.5)),
      child: Container(width: 100, height: 100, color: Colors.blue, child: const Center(child: Text('Drag me'))),
    );
  }
}
