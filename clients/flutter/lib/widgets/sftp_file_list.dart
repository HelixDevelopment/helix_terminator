import 'package:flutter/material.dart';

class SftpFileList extends StatelessWidget {
  final List<dynamic> files;
  final ValueChanged<dynamic>? onFileTap;

  const SftpFileList({super.key, required this.files, this.onFileTap});

  @override
  Widget build(BuildContext context) {
    return ListView.builder(
      itemCount: files.length,
      itemBuilder: (context, index) {
        final file = files[index];
        return ListTile(
          leading: Icon(file.isDirectory ? Icons.folder : Icons.insert_drive_file),
          title: Text(file.name),
          onTap: () => onFileTap?.call(file),
        );
      },
    );
  }
}
