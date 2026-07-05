import 'package:flutter/material.dart';

class MenuScale extends StatelessWidget {
  const MenuScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        PopupMenuButton<String>(
          itemBuilder: (_) => const [
            PopupMenuItem(value: 'a', child: Text('Item 1')),
            PopupMenuItem(value: 'b', child: Text('Item 2')),
          ],
          child: const ElevatedButton(onPressed: null, child: Text('Popup Menu')),
        ),
        DropdownButton<String>(
          items: const [DropdownMenuItem(value: 'a', child: Text('A'))],
          onChanged: (_) {},
          hint: const Text('Dropdown'),
        ),
      ],
    );
  }
}
