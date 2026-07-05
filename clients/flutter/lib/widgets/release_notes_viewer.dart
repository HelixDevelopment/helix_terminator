import 'package:flutter/material.dart';

class ReleaseNotesViewer extends StatelessWidget {
  final String version;
  final String notes;

  const ReleaseNotesViewer({super.key, required this.version, required this.notes});

  @override
  Widget build(BuildContext context) {
    return ExpansionTile(
      title: Text('Version $version'),
      children: [Padding(padding: const EdgeInsets.all(16), child: Text(notes))],
    );
  }
}
