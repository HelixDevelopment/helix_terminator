import 'package:flutter/material.dart';

class SafeAreaScale extends StatelessWidget {
  const SafeAreaScale({super.key});

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      child: Container(color: Colors.blue, child: const Center(child: Text('Safe Area'))),
    );
  }
}
