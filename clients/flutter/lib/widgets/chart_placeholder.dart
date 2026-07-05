import 'package:flutter/material.dart';

class ChartPlaceholder extends StatelessWidget {
  final String title;

  const ChartPlaceholder({super.key, required this.title});

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(title, style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 8),
            Container(
              height: 150,
              color: Colors.grey.shade200,
              child: const Center(child: Text('Chart Placeholder')),
            ),
          ],
        ),
      ),
    );
  }
}
