import 'package:flutter/material.dart';

class BottomSheetScale extends StatelessWidget {
  const BottomSheetScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        ElevatedButton(
          onPressed: () => showModalBottomSheet(
            context: context,
            builder: (_) => Container(height: 200, child: const Center(child: Text('Modal Bottom Sheet'))),
          ),
          child: const Text('Modal Bottom Sheet'),
        ),
        ElevatedButton(
          onPressed: () => showBottomSheet(
            context: context,
            builder: (_) => Container(height: 200, child: const Center(child: Text('Bottom Sheet'))),
          ),
          child: const Text('Bottom Sheet'),
        ),
      ],
    );
  }
}
