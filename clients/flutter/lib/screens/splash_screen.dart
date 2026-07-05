import 'package:flutter/material.dart';

class SplashScreen extends StatelessWidget {
  const SplashScreen({super.key});

  @override
  Widget build(BuildContext context) {
    // TODO: implement splash logic (auth check, navigation)
    return const Scaffold(
      body: Center(child: CircularProgressIndicator()),
    );
  }
}
