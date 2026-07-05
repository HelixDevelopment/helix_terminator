import 'package:flutter/material.dart';

class DividerScale extends StatelessWidget {
  const DividerScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: const [
        Divider(),
        SizedBox(height: 8),
        Divider(thickness: 2),
        SizedBox(height: 8),
        Divider(indent: 16, endIndent: 16),
      ],
    );
  }
}
