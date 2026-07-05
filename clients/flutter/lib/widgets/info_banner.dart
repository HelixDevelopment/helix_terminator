import 'package:flutter/material.dart';

class InfoBanner extends StatelessWidget {
  final String message;
  final Color backgroundColor;

  const InfoBanner({super.key, required this.message, this.backgroundColor = Colors.blue});

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      color: backgroundColor,
      padding: const EdgeInsets.all(12),
      child: Text(message, style: const TextStyle(color: Colors.white)),
    );
  }
}
