import 'package:flutter/material.dart';

class PdfViewerStub extends StatelessWidget {
  final String path;

  const PdfViewerStub({super.key, required this.path});

  @override
  Widget build(BuildContext context) {
    // TODO: integrate pdfx or native_pdf_view
    return Container(
      color: Colors.grey.shade200,
      child: Center(child: Text('PDF: $path')),
    );
  }
}
