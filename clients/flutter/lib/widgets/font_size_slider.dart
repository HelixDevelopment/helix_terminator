import 'package:flutter/material.dart';

class FontSizeSlider extends StatelessWidget {
  final double value;
  final ValueChanged<double>? onChanged;

  const FontSizeSlider({super.key, required this.value, this.onChanged});

  @override
  Widget build(BuildContext context) {
    return Slider(
      value: value,
      min: 8,
      max: 32,
      divisions: 24,
      label: '${value.round()}px',
      onChanged: onChanged,
    );
  }
}
