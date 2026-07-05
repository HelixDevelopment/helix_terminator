import 'package:flutter/material.dart';

class RiveAnimationStub extends StatelessWidget {
  final String assetPath;

  const RiveAnimationStub({super.key, required this.assetPath});

  @override
  Widget build(BuildContext context) {
    // TODO: integrate rive package
    return Container(
      height: 200,
      child: Center(child: Text('Rive: $assetPath')),
    );
  }
}
