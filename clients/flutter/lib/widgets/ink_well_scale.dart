import 'package:flutter/material.dart';

class InkWellScale extends StatelessWidget {
  const InkWellScale({super.key});

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: () {},
      child: Container(width: 100, height: 100, color: Colors.blue, child: const Center(child: Text('InkWell'))),
    );
  }
}
