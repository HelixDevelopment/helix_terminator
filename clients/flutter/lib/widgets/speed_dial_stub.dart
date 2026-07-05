import 'package:flutter/material.dart';

class SpeedDialStub extends StatelessWidget {
  final List<Widget> children;

  const SpeedDialStub({super.key, required this.children});

  @override
  Widget build(BuildContext context) {
    // TODO: integrate flutter_speed_dial
    return FloatingActionButton(
      onPressed: () {},
      child: const Icon(Icons.add),
    );
  }
}
