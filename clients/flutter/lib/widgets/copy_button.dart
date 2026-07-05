import 'package:flutter/material.dart';

class CopyButton extends StatelessWidget {
  final String textToCopy;

  const CopyButton({super.key, required this.textToCopy});

  @override
  Widget build(BuildContext context) {
    return IconButton(
      icon: const Icon(Icons.copy),
      onPressed: () {
        // TODO: implement clipboard copy
      },
    );
  }
}
