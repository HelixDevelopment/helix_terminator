import 'package:flutter/material.dart';

class SshKeyFingerprint extends StatelessWidget {
  final String fingerprint;

  const SshKeyFingerprint({super.key, required this.fingerprint});

  @override
  Widget build(BuildContext context) {
    return SelectableText(
      fingerprint,
      style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
    );
  }
}
