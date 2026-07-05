import 'package:flutter/material.dart';

class PlatformFilePicker extends StatelessWidget {
  final ValueChanged<List<dynamic>>? onFilesPicked;

  const PlatformFilePicker({super.key, this.onFilesPicked});

  @override
  Widget build(BuildContext context) {
    // TODO: implement file_picker integration
    return ElevatedButton(
      onPressed: () {},
      child: const Text('Pick Files'),
    );
  }
}
