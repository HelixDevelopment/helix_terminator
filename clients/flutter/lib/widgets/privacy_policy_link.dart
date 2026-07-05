import 'package:flutter/material.dart';

class PrivacyPolicyLink extends StatelessWidget {
  final VoidCallback? onTap;

  const PrivacyPolicyLink({super.key, this.onTap});

  @override
  Widget build(BuildContext context) {
    return TextButton(
      onPressed: onTap,
      child: const Text('Privacy Policy'),
    );
  }
}
