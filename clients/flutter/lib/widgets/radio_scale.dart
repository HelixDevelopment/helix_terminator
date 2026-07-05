import 'package:flutter/material.dart';

class RadioScale extends StatelessWidget {
  const RadioScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Radio(value: 1, groupValue: 1, onChanged: (_) {}),
        Radio(value: 2, groupValue: 1, onChanged: (_) {}),
      ],
    );
  }
}
