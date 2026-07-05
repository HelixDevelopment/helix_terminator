import 'package:flutter/material.dart';

class SwipeActionList extends StatelessWidget {
  final List<Widget> children;

  const SwipeActionList({super.key, required this.children});

  @override
  Widget build(BuildContext context) {
    // TODO: implement dismissible swipe actions
    return ListView(children: children);
  }
}
