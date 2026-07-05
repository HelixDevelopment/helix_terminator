import 'package:flutter/material.dart';

class OffstageScale extends StatelessWidget {
  const OffstageScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        const Offstage(offstage: false, child: Text('Visible')),
        const Offstage(offstage: true, child: Text('Hidden')),
      ],
    );
  }
}
