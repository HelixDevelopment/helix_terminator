import 'package:flutter/material.dart';

class DataExportButton extends StatelessWidget {
  final VoidCallback? onExport;

  const DataExportButton({super.key, this.onExport});

  @override
  Widget build(BuildContext context) {
    return ElevatedButton.icon(
      onPressed: onExport,
      icon: const Icon(Icons.download),
      label: const Text('Export My Data'),
    );
  }
}
