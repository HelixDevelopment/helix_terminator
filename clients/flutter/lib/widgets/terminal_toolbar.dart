import 'package:flutter/material.dart';

class TerminalToolbar extends StatelessWidget {
  final VoidCallback? onCopy;
  final VoidCallback? onPaste;
  final VoidCallback? onClear;

  const TerminalToolbar({super.key, this.onCopy, this.onPaste, this.onClear});

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        IconButton(icon: const Icon(Icons.copy), onPressed: onCopy),
        IconButton(icon: const Icon(Icons.paste), onPressed: onPaste),
        IconButton(icon: const Icon(Icons.clear), onPressed: onClear),
      ],
    );
  }
}
