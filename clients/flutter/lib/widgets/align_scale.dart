import 'package:flutter/material.dart';

class AlignScale extends StatelessWidget {
  const AlignScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 200,
      height: 200,
      color: Colors.blue,
      child: const Align(
        alignment: Alignment.bottomRight,
        child: Text('Bottom Right'),
      ),
    );
  }
}
