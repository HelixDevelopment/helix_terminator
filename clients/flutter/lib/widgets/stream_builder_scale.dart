import 'package:flutter/material.dart';

class StreamBuilderScale extends StatelessWidget {
  const StreamBuilderScale({super.key});

  @override
  Widget build(BuildContext context) {
    return StreamBuilder<int>(
      stream: Stream.periodic(const Duration(seconds: 1), (i) => i).take(10),
      builder: (context, snapshot) {
        return Text('Count: ${snapshot.data ?? 0}');
      },
    );
  }
}
