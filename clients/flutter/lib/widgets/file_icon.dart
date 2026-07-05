import 'package:flutter/material.dart';

class FileIcon extends StatelessWidget {
  final String extension;
  final double size;

  const FileIcon({super.key, required this.extension, this.size = 40});

  @override
  Widget build(BuildContext context) {
    final icon = switch (extension.toLowerCase()) {
      'pdf' => Icons.picture_as_pdf,
      'jpg' || 'jpeg' || 'png' || 'gif' => Icons.image,
      'mp4' || 'mov' || 'avi' => Icons.video_file,
      'mp3' || 'wav' || 'flac' => Icons.audio_file,
      'zip' || 'tar' || 'gz' || 'rar' => Icons.folder_zip,
      'doc' || 'docx' => Icons.description,
      'xls' || 'xlsx' => Icons.table_chart,
      'ppt' || 'pptx' => Icons.slideshow,
      'txt' || 'md' || 'json' || 'xml' || 'yaml' || 'yml' => Icons.text_snippet,
      'go' || 'py' || 'js' || 'ts' || 'java' || 'cpp' || 'c' || 'rs' || 'dart' => Icons.code,
      _ => Icons.insert_drive_file,
    };
    return Icon(icon, size: size);
  }
}
