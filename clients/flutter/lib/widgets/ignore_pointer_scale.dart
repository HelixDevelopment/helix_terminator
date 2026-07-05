import 'package:flutter/material.dart';

class IgnorePointerScale extends StatelessWidget {
  const IgnorePointerScale({super.key});

  @override
  Widget build(BuildContext context) {
    return IgnorePointer(
      ignoring: true,
      child: ElevatedButton(onPressed: () {}, child: const Text('Ignored')),
    );
  }
}
