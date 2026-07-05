import 'package:flutter/material.dart';

class SnippetCard extends StatelessWidget {
  final String title;
  final String language;
  final VoidCallback? onTap;

  const SnippetCard({super.key, required this.title, required this.language, this.onTap});

  @override
  Widget build(BuildContext context) {
    return Card(
      child: ListTile(
        leading: const Icon(Icons.code),
        title: Text(title),
        subtitle: Text(language),
        onTap: onTap,
      ),
    );
  }
}
