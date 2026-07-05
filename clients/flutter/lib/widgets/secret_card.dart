import 'package:flutter/material.dart';

class SecretCard extends StatelessWidget {
  final String name;
  final String type;
  final VoidCallback? onTap;

  const SecretCard({super.key, required this.name, required this.type, this.onTap});

  @override
  Widget build(BuildContext context) {
    return Card(
      child: ListTile(
        leading: const Icon(Icons.lock),
        title: Text(name),
        subtitle: Text(type),
        onTap: onTap,
      ),
    );
  }
}
