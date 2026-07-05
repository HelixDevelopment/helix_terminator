import 'package:flutter/material.dart';

class IconGrid extends StatelessWidget {
  final List<IconData> icons;

  const IconGrid({super.key, required this.icons});

  @override
  Widget build(BuildContext context) {
    return GridView.builder(
      shrinkWrap: true,
      gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(crossAxisCount: 6),
      itemCount: icons.length,
      itemBuilder: (context, index) => Icon(icons[index]),
    );
  }
}
