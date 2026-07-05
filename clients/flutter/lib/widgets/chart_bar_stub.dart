import 'package:flutter/material.dart';

class ChartBarStub extends StatelessWidget {
  final List<double> data;

  const ChartBarStub({super.key, required this.data});

  @override
  Widget build(BuildContext context) {
    // TODO: integrate fl_chart
    return Container(
      height: 200,
      color: Colors.grey.shade200,
      child: const Center(child: Text('Bar Chart')),
    );
  }
}
