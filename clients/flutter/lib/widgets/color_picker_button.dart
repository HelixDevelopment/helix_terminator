import 'package:flutter/material.dart';

class ColorPickerButton extends StatelessWidget {
  final Color color;
  final ValueChanged<Color>? onColorChanged;

  const ColorPickerButton({super.key, required this.color, this.onColorChanged});

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: () {
        // TODO: show color picker dialog
      },
      child: Container(
        width: 40,
        height: 40,
        decoration: BoxDecoration(color: color, borderRadius: BorderRadius.circular(8)),
      ),
    );
  }
}
