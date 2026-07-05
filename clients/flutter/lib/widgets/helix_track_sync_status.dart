import 'package:flutter/material.dart';

class HelixTrackSyncStatus extends StatelessWidget {
  final bool synced;
  final DateTime? lastSync;

  const HelixTrackSyncStatus({super.key, required this.synced, this.lastSync});

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(synced ? Icons.sync : Icons.sync_disabled, color: synced ? Colors.green : Colors.orange),
        const SizedBox(width: 4),
        Text(lastSync != null ? 'Synced ${lastSync!.toIso8601String()}' : 'Never synced'),
      ],
    );
  }
}
