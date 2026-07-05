import 'package:flutter/material.dart';

class PersistentBottomSheetScale extends StatelessWidget {
  const PersistentBottomSheetScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Builder(
        builder: (context) => Center(
          child: ElevatedButton(
            onPressed: () {
              showBottomSheet(
                context: context,
                builder: (_) => Container(
                  height: 200,
                  color: Colors.blue,
                  child: const Center(child: Text('Persistent Bottom Sheet')),
                ),
              );
            },
            child: const Text('Show Bottom Sheet'),
          ),
        ),
      ),
    );
  }
}
