import 'package:flutter/material.dart';

class FindReplaceBar extends StatelessWidget {
  final TextEditingController? findController;
  final TextEditingController? replaceController;
  final VoidCallback? onFind;
  final VoidCallback? onReplace;

  const FindReplaceBar({
    super.key,
    this.findController,
    this.replaceController,
    this.onFind,
    this.onReplace,
  });

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(
          child: TextField(
            controller: findController,
            decoration: const InputDecoration(hintText: 'Find'),
          ),
        ),
        Expanded(
          child: TextField(
            controller: replaceController,
            decoration: const InputDecoration(hintText: 'Replace'),
          ),
        ),
        IconButton(icon: const Icon(Icons.search), onPressed: onFind),
        IconButton(icon: const Icon(Icons.find_replace), onPressed: onReplace),
      ],
    );
  }
}
