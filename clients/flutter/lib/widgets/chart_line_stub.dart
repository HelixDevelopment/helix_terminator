import 'package:flutter/material.dart';

class ChartLineStub extends StatelessWidget {
  final List<double> data;

  const ChartLineStub({super.key, required this.data});

  @override
  Widget build(BuildContext context) {
    // TODO: integrate fl_chart
    return Container(
      height: 200,
      color: Colors.grey.shade200,
      child: const Center(child: Text('Line Chart')),
    );
  }
}
