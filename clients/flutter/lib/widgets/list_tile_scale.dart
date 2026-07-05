import 'package:flutter/material.dart';

class ListTileScale extends StatelessWidget {
  const ListTileScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        const ListTile(title: Text('One-line')),
        const ListTile(leading: Icon(Icons.person), title: Text('With leading')),
        const ListTile(title: Text('With trailing'), trailing: Icon(Icons.arrow_forward)),
        ListTile(
          title: const Text('With subtitle'),
          subtitle: const Text('Subtitle text'),
          onTap: () {},
        ),
      ],
    );
  }
}
