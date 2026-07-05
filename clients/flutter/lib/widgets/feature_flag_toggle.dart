import 'package:flutter/material.dart';

class FeatureFlagToggle extends StatelessWidget {
  final String name;
  final bool enabled;
  final ValueChanged<bool>? onChanged;

  const FeatureFlagToggle({super.key, required this.name, required this.enabled, this.onChanged});

  @override
  Widget build(BuildContext context) {
    return SwitchListTile(
      title: Text(name),
      value: enabled,
      onChanged: onChanged,
    );
  }
}
