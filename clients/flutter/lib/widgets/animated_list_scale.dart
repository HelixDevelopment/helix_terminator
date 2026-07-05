import 'package:flutter/material.dart';

class AnimatedListScale extends StatelessWidget {
  const AnimatedListScale({super.key});

  @override
  Widget build(BuildContext context) {
    return AnimatedList(
      initialItemCount: 3,
      itemBuilder: (context, index, animation) {
        return SizeTransition(
          sizeFactor: animation,
          child: ListTile(title: Text('Item $index')),
        );
      },
    );
  }
}
