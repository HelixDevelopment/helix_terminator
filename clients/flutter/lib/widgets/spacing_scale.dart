import 'package:flutter/material.dart';

class SpacingScale extends StatelessWidget {
  const SpacingScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [4, 8, 12, 16, 24, 32, 48, 64].map((s) {
        return Row(
          children: [
            Text('$s px'),
            const SizedBox(width: 8),
            Container(width: s.toDouble(), height: 16, color: Colors.blue),
          ],
        );
      }).toList(),
    );
  }
}
