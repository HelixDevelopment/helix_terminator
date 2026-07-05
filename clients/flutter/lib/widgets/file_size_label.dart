import 'package:flutter/material.dart';

class FileSizeLabel extends StatelessWidget {
  final int bytes;

  const FileSizeLabel({super.key, required this.bytes});

  @override
  Widget build(BuildContext context) {
    String size;
    if (bytes >= 1024 * 1024 * 1024) {
      size = '${(bytes / (1024 * 1024 * 1024)).toStringAsFixed(2)} GB';
    } else if (bytes >= 1024 * 1024) {
      size = '${(bytes / (1024 * 1024)).toStringAsFixed(2)} MB';
    } else if (bytes >= 1024) {
      size = '${(bytes / 1024).toStringAsFixed(2)} KB';
    } else {
      size = '$bytes B';
    }
    return Text(size, style: Theme.of(context).textTheme.bodySmall);
  }
}
