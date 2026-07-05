import 'package:flutter/material.dart';

class FutureBuilderScale extends StatelessWidget {
  const FutureBuilderScale({super.key});

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<String>(
      future: Future.delayed(const Duration(seconds: 1), () => 'Loaded'),
      builder: (context, snapshot) {
        if (snapshot.connectionState == ConnectionState.waiting) {
          return const CircularProgressIndicator();
        }
        return Text(snapshot.data ?? '');
      },
    );
  }
}
