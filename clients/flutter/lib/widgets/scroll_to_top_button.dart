import 'package:flutter/material.dart';

class ScrollToTopButton extends StatelessWidget {
  final ScrollController controller;

  const ScrollToTopButton({super.key, required this.controller});

  @override
  Widget build(BuildContext context) {
    return FloatingActionButton.small(
      onPressed: () => controller.animateTo(0, duration: const Duration(milliseconds: 300), curve: Curves.easeOut),
      child: const Icon(Icons.arrow_upward),
    );
  }
}
