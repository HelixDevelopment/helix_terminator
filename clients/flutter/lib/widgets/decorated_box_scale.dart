import 'package:flutter/material.dart';

class DecoratedBoxScale extends StatelessWidget {
  const DecoratedBoxScale({super.key});

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: BoxDecoration(
        color: Colors.blue,
        borderRadius: BorderRadius.circular(16),
        boxShadow: [BoxShadow(blurRadius: 8, color: Colors.black.withOpacity(0.2))],
      ),
      child: const SizedBox(width: 100, height: 100, child: Center(child: Text('Decorated'))),
    );
  }
}
