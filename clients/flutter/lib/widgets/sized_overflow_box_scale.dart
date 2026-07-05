import 'package:flutter/material.dart';

class SizedOverflowBoxScale extends StatelessWidget {
  const SizedOverflowBoxScale({super.key});

  @override
  Widget build(BuildContext context) {
    return SizedOverflowBox(
      size: const Size(100, 100),
      child: Container(width: 150, height: 150, color: Colors.blue.withOpacity(0.5)),
    );
  }
}
