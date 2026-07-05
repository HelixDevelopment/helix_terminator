import 'package:flutter/material.dart';

class BadgeIcon extends StatelessWidget {
  final IconData icon;
  final int count;
  final Color badgeColor;

  const BadgeIcon({super.key, required this.icon, this.count = 0, this.badgeColor = Colors.red});

  @override
  Widget build(BuildContext context) {
    return Badge(
      isLabelVisible: count > 0,
      label: Text('$count'),
      backgroundColor: badgeColor,
      child: Icon(icon),
    );
  }
}
