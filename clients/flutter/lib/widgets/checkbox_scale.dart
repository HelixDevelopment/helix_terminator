import 'package:flutter/material.dart';

class CheckboxScale extends StatelessWidget {
  const CheckboxScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Checkbox(value: true, onChanged: (_) {}),
        Checkbox(value: false, onChanged: (_) {}),
        Checkbox(value: null, tristate: true, onChanged: (_) {}),
      ],
    );
  }
}
