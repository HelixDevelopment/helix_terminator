import 'package:flutter/material.dart';

class SliderScale extends StatelessWidget {
  const SliderScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Slider(value: 0.5, onChanged: (_) {}),
        RangeSlider(values: const RangeValues(0.2, 0.8), onChanged: (_) {}),
      ],
    );
  }
}
