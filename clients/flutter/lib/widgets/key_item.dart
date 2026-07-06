import 'package:flutter/material.dart';

class KeyItem extends StatelessWidget {
  final String name;
  final String fingerprint;
  final VoidCallback? onTap;

  const KeyItem({
    super.key,
    required this.name,
    required this.fingerprint,
    this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return ListTile(
      leading: Container(
        width: 40,
        height: 40,
        decoration: BoxDecoration(
          color: theme.colorScheme.primaryContainer,
          borderRadius: BorderRadius.circular(8),
        ),
        child: Icon(
          Icons.vpn_key,
          color: theme.colorScheme.onPrimaryContainer,
        ),
      ),
      title: Text(
        name,
        style: theme.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w600),
      ),
      subtitle: Text(
        fingerprint,
        style: theme.textTheme.bodySmall?.copyWith(
          color: theme.colorScheme.onSurface.withOpacity(0.6),
        ),
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
      ),
      onTap: onTap,
    );
  }
}
