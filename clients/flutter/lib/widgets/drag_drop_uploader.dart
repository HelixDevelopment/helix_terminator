import 'package:flutter/material.dart';

class DragDropUploader extends StatelessWidget {
  final ValueChanged<List<dynamic>>? onFilesDropped;

  const DragDropUploader({super.key, this.onFilesDropped});

  @override
  Widget build(BuildContext context) {
    // TODO: implement desktop drag-and-drop
    return Container(
      height: 120,
      decoration: BoxDecoration(
        border: Border.all(color: Colors.grey),
        borderRadius: BorderRadius.circular(8),
      ),
      child: const Center(child: Text('Drop files here')),
    );
  }
}
