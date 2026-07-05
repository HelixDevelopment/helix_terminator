import 'package:flutter/material.dart';

class DraggableScrollableSheetScale extends StatelessWidget {
  const DraggableScrollableSheetScale({super.key});

  @override
  Widget build(BuildContext context) {
    return DraggableScrollableSheet(
      builder: (context, scrollController) {
        return Container(
          color: Colors.blue,
          child: ListView.builder(
            controller: scrollController,
            itemCount: 20,
            itemBuilder: (context, index) => ListTile(title: Text('Item $index')),
          ),
        );
      },
    );
  }
}
