import 'package:flutter/material.dart';

class OverflowBoxScale extends StatelessWidget {
  const OverflowBoxScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 50,
      height: 50,
      color: Colors.red,
      child: OverflowBox(
        maxWidth: 100,
        maxHeight: 100,
        child: Container(width: 100, height: 100, color: Colors.blue.withOpacity(0.5)),
      ),
    );
  }
}
