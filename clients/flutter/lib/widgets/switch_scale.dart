import 'package:flutter/material.dart';

class SwitchScale extends StatelessWidget {
  const SwitchScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Switch(value: true, onChanged: (_) {}),
        Switch(value: false, onChanged: (_) {}),
        Switch.adaptive(value: true, onChanged: (_) {}),
      ],
    );
  }
}
