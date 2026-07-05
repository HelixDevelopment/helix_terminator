import 'package:flutter/material.dart';

class TabSwitcher extends StatelessWidget {
  final List<String> tabs;
  final int activeIndex;
  final ValueChanged<int>? onTap;

  const TabSwitcher({super.key, required this.tabs, required this.activeIndex, this.onTap});

  @override
  Widget build(BuildContext context) {
    return Row(
      children: tabs.asMap().entries.map((e) {
        return Expanded(
          child: InkWell(
            onTap: () => onTap?.call(e.key),
            child: Container(
              padding: const EdgeInsets.symmetric(vertical: 12),
              decoration: BoxDecoration(
                border: Border(
                  bottom: BorderSide(
                    color: e.key == activeIndex ? Theme.of(context).colorScheme.primary : Colors.transparent,
                    width: 2,
                  ),
                ),
              ),
              child: Text(e.value, textAlign: TextAlign.center),
            ),
          ),
        );
      }).toList(),
    );
  }
}
