import 'package:flutter/material.dart';

class AdaptiveListView extends StatelessWidget {
  final List<Widget> children;

  const AdaptiveListView({super.key, required this.children});

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        if (constraints.maxWidth >= 1200) {
          return GridView.count(
            crossAxisCount: 3,
            children: children,
          );
        }
        if (constraints.maxWidth >= 600) {
          return GridView.count(
            crossAxisCount: 2,
            children: children,
          );
        }
        return ListView(children: children);
      },
    );
  }
}
