import 'package:flutter/material.dart';

class GanttChartStub extends StatelessWidget {
  final List<dynamic> tasks;

  const GanttChartStub({super.key, required this.tasks});

  @override
  Widget build(BuildContext context) {
    // TODO: integrate flutter_gantt_chart or custom implementation
    return Container(
      height: 200,
      color: Colors.grey.shade200,
      child: const Center(child: Text('Gantt Chart')),
    );
  }
}
