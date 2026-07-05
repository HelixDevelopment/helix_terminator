import 'package:flutter/material.dart';

class FractionallySizedBoxScale extends StatelessWidget {
  const FractionallySizedBoxScale({super.key});

  @override
  Widget build(BuildContext context) {
    return FractionallySizedBox(
      widthFactor: 0.5,
      heightFactor: 0.5,
      child: Container(color: Colors.blue, child: const Center(child: Text('50%'))),
    );
  }
}
