import 'package:flutter/material.dart';

class LottieAnimationStub extends StatelessWidget {
  final String assetPath;

  const LottieAnimationStub({super.key, required this.assetPath});

  @override
  Widget build(BuildContext context) {
    // TODO: integrate lottie package
    return Container(
      height: 200,
      child: Center(child: Text('Lottie: $assetPath')),
    );
  }
}
