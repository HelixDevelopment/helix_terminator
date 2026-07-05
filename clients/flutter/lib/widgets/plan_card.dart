import 'package:flutter/material.dart';

class PlanCard extends StatelessWidget {
  final String name;
  final String price;
  final List<String> features;
  final bool isCurrent;
  final VoidCallback? onSelect;

  const PlanCard({
    super.key,
    required this.name,
    required this.price,
    required this.features,
    this.isCurrent = false,
    this.onSelect,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(name, style: Theme.of(context).textTheme.headlineSmall),
            Text(price, style: Theme.of(context).textTheme.titleLarge),
            const Divider(),
            ...features.map((f) => ListTile(leading: const Icon(Icons.check), title: Text(f), dense: true)),
            if (isCurrent)
              const Chip(label: Text('Current Plan'))
            else
              ElevatedButton(onPressed: onSelect, child: const Text('Select')),
          ],
        ),
      ),
    );
  }
}
