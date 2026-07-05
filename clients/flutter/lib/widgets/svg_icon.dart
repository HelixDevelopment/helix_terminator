import 'package:flutter/material.dart';

class SvgIcon extends StatelessWidget {
  final String assetPath;
  final double size;
  final Color? color;

  const SvgIcon({super.key, required this.assetPath, this.size = 24, this.color});

  @override
  Widget build(BuildContext context) {
    // TODO: integrate flutter_svg
    return Icon(Icons.image, size: size, color: color);
  }
}
