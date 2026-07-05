import 'package:flutter/material.dart';

class TOTPInput extends StatelessWidget {
  final TextEditingController? controller;
  final ValueChanged<String>? onCompleted;

  const TOTPInput({super.key, this.controller, this.onCompleted});

  @override
  Widget build(BuildContext context) {
    return TextField(
      controller: controller,
      maxLength: 6,
      keyboardType: TextInputType.number,
      textAlign: TextAlign.center,
      decoration: const InputDecoration(hintText: '000000', counterText: ''),
      onChanged: (v) {
        if (v.length == 6) onCompleted?.call(v);
      },
    );
  }
}
