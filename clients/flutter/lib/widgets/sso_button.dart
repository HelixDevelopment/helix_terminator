import 'package:flutter/material.dart';

class SsoButton extends StatelessWidget {
  final String provider;
  final VoidCallback? onTap;

  const SsoButton({super.key, required this.provider, this.onTap});

  @override
  Widget build(BuildContext context) {
    return ElevatedButton.icon(
      onPressed: onTap,
      icon: const Icon(Icons.login),
      label: Text('Sign in with $provider'),
    );
  }
}
