import 'package:flutter/material.dart';

class ExpandableCard extends StatelessWidget {
  final String title;
  final Widget child;

  const ExpandableCard({super.key, required this.title, required this.child});

  @override
  Widget build(BuildContext context) {
    return ExpansionTile(
      title: Text(title),
      children: [child],
    );
  }
}
