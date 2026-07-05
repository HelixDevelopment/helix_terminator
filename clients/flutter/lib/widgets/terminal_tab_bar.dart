import 'package:flutter/material.dart';

class TerminalTabBar extends StatelessWidget {
  final List<String> tabs;
  final int activeIndex;
  final ValueChanged<int>? onTap;
  final ValueChanged<int>? onClose;

  const TerminalTabBar({super.key, required this.tabs, required this.activeIndex, this.onTap, this.onClose});

  @override
  Widget build(BuildContext context) {
    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      child: Row(
        children: tabs.asMap().entries.map((e) {
          return GestureDetector(
            onTap: () => onTap?.call(e.key),
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
              decoration: BoxDecoration(
                color: e.key == activeIndex ? Theme.of(context).colorScheme.primaryContainer : null,
                border: const Border(bottom: BorderSide(color: Colors.grey)),
              ),
              child: Row(
                children: [
                  Text(e.value),
                  const SizedBox(width: 4),
                  InkWell(
                    onTap: () => onClose?.call(e.key),
                    child: const Icon(Icons.close, size: 14),
                  ),
                ],
              ),
            ),
          );
        }).toList(),
      ),
    );
  }
}
