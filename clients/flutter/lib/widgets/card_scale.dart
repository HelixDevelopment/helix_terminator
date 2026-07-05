import 'package:flutter/material.dart';

class CardScale extends StatelessWidget {
  const CardScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Card(child: ListTile(title: const Text('Card'))),
        const Card.filled(child: ListTile(title: Text('Filled Card'))),
        const Card.outlined(child: ListTile(title: Text('Outlined Card'))),
      ],
    );
  }
}
