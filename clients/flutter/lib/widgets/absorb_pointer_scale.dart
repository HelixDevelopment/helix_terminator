import 'package:flutter/material.dart';

class AbsorbPointerScale extends StatelessWidget {
  const AbsorbPointerScale({super.key});

  @override
  Widget build(BuildContext context) {
    return AbsorbPointer(
      absorbing: true,
      child: ElevatedButton(onPressed: () {}, child: const Text('Absorbed')),
    );
  }
}
