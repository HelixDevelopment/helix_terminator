import 'package:flutter/material.dart';

class HoverReveal extends StatefulWidget {
  final Widget child;
  final Widget revealed;

  const HoverReveal({super.key, required this.child, required this.revealed});

  @override
  State<HoverReveal> createState() => _HoverRevealState();
}

class _HoverRevealState extends State<HoverReveal> {
  bool _hover = false;

  @override
  Widget build(BuildContext context) {
    return MouseRegion(
      onEnter: (_) => setState(() => _hover = true),
      onExit: (_) => setState(() => _hover = false),
      child: Stack(
        children: [
          widget.child,
          if (_hover) widget.revealed,
        ],
      ),
    );
  }
}
