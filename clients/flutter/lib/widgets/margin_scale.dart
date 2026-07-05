import 'package:flutter/material.dart';

class MarginScale extends StatelessWidget {
  const MarginScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Container(color: Colors.blue, child: const Text('No margin')),
        Container(color: Colors.red, child: Container(color: Colors.blue, margin: const EdgeInsets.all(16), child: const Text('Margin 16'))),
      ],
    );
  }
}
