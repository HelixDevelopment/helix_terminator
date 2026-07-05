import 'package:flutter/material.dart';

class HeatmapCalendarStub extends StatelessWidget {
  final Map<DateTime, int> data;

  const HeatmapCalendarStub({super.key, required this.data});

  @override
  Widget build(BuildContext context) {
    // TODO: integrate heatmap_calendar or flutter_heatmap_calendar
    return Container(
      height: 200,
      color: Colors.grey.shade200,
      child: const Center(child: Text('Heatmap Calendar')),
    );
  }
}
