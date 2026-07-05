import 'package:flutter/material.dart';

class ColorSchemePreview extends StatelessWidget {
  final List<Color> colors;

  const ColorSchemePreview({super.key, required this.colors});

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 8,
      children: colors.map((c) => Container(width: 40, height: 40, color: c)).toList(),
    );
  }
}
