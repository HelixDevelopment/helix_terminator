import 'package:flutter/material.dart';

class TooltipIcon extends StatelessWidget {
  final IconData icon;
  final String tooltip;
  final VoidCallback? onTap;

  const TooltipIcon({super.key, required this.icon, required this.tooltip, this.onTap});

  @override
  Widget build(BuildContext context) {
    return Tooltip(
      message: tooltip,
      child: IconButton(icon: Icon(icon), onPressed: onTap),
    );
  }
}
