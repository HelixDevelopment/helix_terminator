import 'package:flutter/material.dart';

class TypographyScale extends StatelessWidget {
  const TypographyScale({super.key});

  @override
  Widget build(BuildContext context) {
    final styles = [
      Theme.of(context).textTheme.displayLarge,
      Theme.of(context).textTheme.displayMedium,
      Theme.of(context).textTheme.displaySmall,
      Theme.of(context).textTheme.headlineLarge,
      Theme.of(context).textTheme.headlineMedium,
      Theme.of(context).textTheme.headlineSmall,
      Theme.of(context).textTheme.titleLarge,
      Theme.of(context).textTheme.titleMedium,
      Theme.of(context).textTheme.titleSmall,
      Theme.of(context).textTheme.bodyLarge,
      Theme.of(context).textTheme.bodyMedium,
      Theme.of(context).textTheme.bodySmall,
      Theme.of(context).textTheme.labelLarge,
      Theme.of(context).textTheme.labelMedium,
      Theme.of(context).textTheme.labelSmall,
    ];
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: styles.map((s) => Text('Typography', style: s)).toList(),
    );
  }
}
