import 'package:flutter/material.dart';

class DismissibleListScale extends StatelessWidget {
  const DismissibleListScale({super.key});

  @override
  Widget build(BuildContext context) {
    return ListView.builder(
      itemCount: 5,
      itemBuilder: (context, index) {
        return Dismissible(
          key: ValueKey(index),
          onDismissed: (_) {},
          background: Container(color: Colors.red),
          child: ListTile(title: Text('Item $index')),
        );
      },
    );
  }
}
