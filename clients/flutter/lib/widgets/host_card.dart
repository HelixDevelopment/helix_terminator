import 'package:flutter/material.dart';

class HostCard extends StatelessWidget {
  final String name;
  final String address;
  final VoidCallback? onTap;

  const HostCard({super.key, required this.name, required this.address, this.onTap});

  @override
  Widget build(BuildContext context) {
    return Card(
      child: ListTile(
        leading: const Icon(Icons.computer),
        title: Text(name),
        subtitle: Text(address),
        onTap: onTap,
      ),
    );
  }
}
