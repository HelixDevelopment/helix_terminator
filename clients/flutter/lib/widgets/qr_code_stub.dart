import 'package:flutter/material.dart';

class QRCodeStub extends StatelessWidget {
  final String data;

  const QRCodeStub({super.key, required this.data});

  @override
  Widget build(BuildContext context) {
    // TODO: integrate qr_flutter
    return Container(
      width: 200,
      height: 200,
      color: Colors.white,
      child: Center(child: Text('QR: $data')),
    );
  }
}
