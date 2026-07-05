import 'package:flutter/material.dart';

class ShadowScale extends StatelessWidget {
  const ShadowScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [1, 2, 3, 4, 5].map((level) {
        return Container(
          margin: const EdgeInsets.all(8),
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: Colors.white,
            boxShadow: [
              BoxShadow(
                blurRadius: level * 4.0,
                spreadRadius: level * 1.0,
                color: Colors.black.withOpacity(0.1),
              ),
            ],
          ),
          child: Text('Shadow level $level'),
        );
      }).toList(),
    );
  }
}
