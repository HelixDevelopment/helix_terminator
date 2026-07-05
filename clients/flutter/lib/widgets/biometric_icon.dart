import 'package:flutter/material.dart';

class BiometricIcon extends StatelessWidget {
  final bool available;

  const BiometricIcon({super.key, required this.available});

  @override
  Widget build(BuildContext context) {
    return Icon(
      available ? Icons.fingerprint : Icons.lock,
      color: available ? Colors.green : Colors.grey,
    );
  }
}
