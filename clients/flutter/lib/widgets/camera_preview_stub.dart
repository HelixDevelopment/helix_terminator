import 'package:flutter/material.dart';

class CameraPreviewStub extends StatelessWidget {
  final VoidCallback? onCapture;

  const CameraPreviewStub({super.key, this.onCapture});

  @override
  Widget build(BuildContext context) {
    // TODO: integrate camera plugin
    return Container(
      color: Colors.black,
      child: Center(
        child: ElevatedButton(onPressed: onCapture, child: const Text('Capture')),
      ),
    );
  }
}
