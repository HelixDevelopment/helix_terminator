import 'package:flutter/material.dart';

class ClipScale extends StatelessWidget {
  const ClipScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        ClipOval(child: Container(width: 100, height: 100, color: Colors.blue)),
        ClipRRect(borderRadius: BorderRadius.circular(16), child: Container(width: 100, height: 100, color: Colors.red)),
      ],
    );
  }
}
