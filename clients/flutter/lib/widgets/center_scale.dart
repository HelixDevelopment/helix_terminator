import 'package:flutter/material.dart';

class CenterScale extends StatelessWidget {
  const CenterScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 200,
      height: 200,
      color: Colors.blue,
      child: const Center(child: Text('Center')),
    );
  }
}
