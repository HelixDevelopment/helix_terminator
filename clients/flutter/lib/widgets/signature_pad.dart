import 'package:flutter/material.dart';

class SignaturePad extends StatelessWidget {
  final VoidCallback? onClear;
  final ValueChanged<dynamic>? onSave;

  const SignaturePad({super.key, this.onClear, this.onSave});

  @override
  Widget build(BuildContext context) {
    // TODO: integrate signature_pad or hand_signature
    return Container(
      height: 200,
      color: Colors.grey.shade200,
      child: const Center(child: Text('Signature Pad')),
    );
  }
}
