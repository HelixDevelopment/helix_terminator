import 'package:flutter/material.dart';

class ChartPieStub extends StatelessWidget {
  final Map<String, double> data;

  const ChartPieStub({super.key, required this.data});

  @override
  Widget build(BuildContext context) {
    // TODO: integrate fl_chart
    return Container(
      height: 200,
      color: Colors.grey.shade200,
      child: const Center(child: Text('Pie Chart')),
    );
  }
}
