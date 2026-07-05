import 'package:flutter/material.dart';

class HostHealthIndicator extends StatelessWidget {
  final bool healthy;
  final int? latencyMs;

  const HostHealthIndicator({super.key, required this.healthy, this.latencyMs});

  @override
  Widget build(BuildContext context) {
    return Tooltip(
      message: latencyMs != null ? '${latencyMs}ms' : 'Unknown',
      child: Icon(
        Icons.circle,
        size: 10,
        color: healthy ? Colors.green : Colors.red,
      ),
    );
  }
}
