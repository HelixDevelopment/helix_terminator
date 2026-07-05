import 'package:flutter/material.dart';

class DesktopWindowControls extends StatelessWidget {
  final VoidCallback? onMinimize;
  final VoidCallback? onMaximize;
  final VoidCallback? onClose;

  const DesktopWindowControls({super.key, this.onMinimize, this.onMaximize, this.onClose});

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        IconButton(icon: const Icon(Icons.remove), onPressed: onMinimize),
        IconButton(icon: const Icon(Icons.crop_square), onPressed: onMaximize),
        IconButton(icon: const Icon(Icons.close), onPressed: onClose),
      ],
    );
  }
}
