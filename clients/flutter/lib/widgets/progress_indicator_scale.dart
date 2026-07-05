import 'package:flutter/material.dart';

class ProgressIndicatorScale extends StatelessWidget {
  const ProgressIndicatorScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: const [
        LinearProgressIndicator(),
        SizedBox(height: 8),
        CircularProgressIndicator(),
        SizedBox(height: 8),
        LinearProgressIndicator(value: 0.5),
      ],
    );
  }
}
