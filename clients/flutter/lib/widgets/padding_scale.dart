import 'package:flutter/material.dart';

class PaddingScale extends StatelessWidget {
  const PaddingScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Container(color: Colors.blue, child: const Text('No padding')),
        Container(color: Colors.blue, child: const Padding(padding: EdgeInsets.all(16), child: Text('Padding 16'))),
      ],
    );
  }
}
