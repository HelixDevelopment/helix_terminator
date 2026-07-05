import 'package:flutter/material.dart';

class TermsOfServiceLink extends StatelessWidget {
  final VoidCallback? onTap;

  const TermsOfServiceLink({super.key, this.onTap});

  @override
  Widget build(BuildContext context) {
    return TextButton(
      onPressed: onTap,
      child: const Text('Terms of Service'),
    );
  }
}
