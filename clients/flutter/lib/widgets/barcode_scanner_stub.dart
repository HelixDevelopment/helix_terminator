import 'package:flutter/material.dart';

class BarcodeScannerStub extends StatelessWidget {
  final ValueChanged<String>? onScanned;

  const BarcodeScannerStub({super.key, this.onScanned});

  @override
  Widget build(BuildContext context) {
    // TODO: integrate mobile_scanner
    return ElevatedButton(
      onPressed: () => onScanned?.call('scanned-data'),
      child: const Text('Scan Barcode'),
    );
  }
}
