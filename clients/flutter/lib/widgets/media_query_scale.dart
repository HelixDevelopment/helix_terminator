import 'package:flutter/material.dart';

class MediaQueryScale extends StatelessWidget {
  const MediaQueryScale({super.key});

  @override
  Widget build(BuildContext context) {
    final size = MediaQuery.of(context).size;
    return Text('Width: ${size.width}, Height: ${size.height}');
  }
}
