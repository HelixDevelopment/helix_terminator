import 'package:flutter/material.dart';

class SegmentedButtonScale extends StatelessWidget {
  const SegmentedButtonScale({super.key});

  @override
  Widget build(BuildContext context) {
    return SegmentedButton<String>(
      segments: const [
        ButtonSegment(value: 'day', label: Text('Day')),
        ButtonSegment(value: 'week', label: Text('Week')),
        ButtonSegment(value: 'month', label: Text('Month')),
      ],
      selected: const {'day'},
      onSelectionChanged: (_) {},
    );
  }
}
