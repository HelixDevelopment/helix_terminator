import 'package:flutter/material.dart';

class FileTransferProgress extends StatelessWidget {
  final String fileName;
  final double progress;
  final int bytesTransferred;
  final int totalBytes;

  const FileTransferProgress({
    super.key,
    required this.fileName,
    required this.progress,
    required this.bytesTransferred,
    required this.totalBytes,
  });

  @override
  Widget build(BuildContext context) {
    return ListTile(
      leading: const Icon(Icons.file_upload),
      title: Text(fileName),
      subtitle: LinearProgressIndicator(value: progress),
      trailing: Text('$bytesTransferred / $totalBytes'),
    );
  }
}
