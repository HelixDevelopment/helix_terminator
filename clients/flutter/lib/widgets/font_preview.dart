import 'package:flutter/material.dart';

class FontPreview extends StatelessWidget {
  final String fontFamily;
  final String sampleText;

  const FontPreview({super.key, required this.fontFamily, this.sampleText = 'The quick brown fox'});

  @override
  Widget build(BuildContext context) {
    return Text(
      sampleText,
      style: TextStyle(fontFamily: fontFamily, fontSize: 18),
    );
  }
}
