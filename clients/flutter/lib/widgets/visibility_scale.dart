import 'package:flutter/material.dart';

class VisibilityScale extends StatelessWidget {
  const VisibilityScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        const Visibility(visible: true, child: Text('Visible')),
        const Visibility(visible: false, child: Text('Hidden')),
      ],
    );
  }
}
