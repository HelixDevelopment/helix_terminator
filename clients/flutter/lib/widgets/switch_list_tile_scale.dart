import 'package:flutter/material.dart';

class SwitchListTileScale extends StatelessWidget {
  const SwitchListTileScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        SwitchListTile(
          value: true,
          onChanged: (_) {},
          title: const Text('On'),
        ),
        SwitchListTile(
          value: false,
          onChanged: (_) {},
          title: const Text('Off'),
        ),
      ],
    );
  }
}
