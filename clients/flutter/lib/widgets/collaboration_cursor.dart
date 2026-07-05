import 'package:flutter/material.dart';

class CollaborationCursor extends StatelessWidget {
  final String userName;
  final Color color;

  const CollaborationCursor({super.key, required this.userName, required this.color});

  @override
  Widget build(BuildContext context) {
    return Tooltip(
      message: userName,
      child: Icon(Icons.arrow_drop_up, color: color, size: 24),
    );
  }
}
