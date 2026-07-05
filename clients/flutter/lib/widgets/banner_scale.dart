import 'package:flutter/material.dart';

class BannerScale extends StatelessWidget {
  const BannerScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        MaterialBanner(
          content: const Text('Default banner'),
          actions: [TextButton(onPressed: () {}, child: const Text('Dismiss'))],
        ),
      ],
    );
  }
}
