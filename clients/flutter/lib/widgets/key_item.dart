import 'package:flutter/material.dart';

class KeyItem extends StatelessWidget {
  final String name;
  final String fingerprint;
  final VoidCallback? onTap;

  const KeyItem({super.key, required this.name, required this.fingerprint, this.onTap});

  @override
  Widget build(BuildContext context) {
    return ListTile(
      leading: const Icon(Icons.vpn_key),
      title: Text(name),
      subtitle: Text(fingerprint),
      onTap: onTap,
    );
  }
}
