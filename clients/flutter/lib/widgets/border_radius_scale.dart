import 'package:flutter/material.dart';

class BorderRadiusScale extends StatelessWidget {
  const BorderRadiusScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 8,
      children: [0, 4, 8, 12, 16, 24].map((r) {
        return Container(
          width: 60,
          height: 60,
          decoration: BoxDecoration(
            color: Colors.blue,
            borderRadius: BorderRadius.circular(r.toDouble()),
          ),
          child: Center(child: Text('$r', style: const TextStyle(color: Colors.white))),
        );
      }).toList(),
    );
  }
}
