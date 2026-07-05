import 'package:flutter/material.dart';

class SyncIndicator extends StatelessWidget {
  final bool isSyncing;

  const SyncIndicator({super.key, required this.isSyncing});

  @override
  Widget build(BuildContext context) {
    return isSyncing
        ? const SizedBox(
            width: 16,
            height: 16,
            child: CircularProgressIndicator(strokeWidth: 2),
          )
        : const Icon(Icons.cloud_done, size: 16);
  }
}
