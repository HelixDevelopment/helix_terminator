import 'package:flutter/material.dart';

class HeroAnimationScale extends StatelessWidget {
  const HeroAnimationScale({super.key});

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: () => Navigator.of(context).push(
        MaterialPageRoute(
          builder: (_) => Scaffold(
            appBar: AppBar(),
            body: Center(
              child: Hero(tag: 'hero', child: Container(width: 200, height: 200, color: Colors.blue)),
            ),
          ),
        ),
      ),
      child: Hero(
        tag: 'hero',
        child: Container(width: 100, height: 100, color: Colors.blue),
      ),
    );
  }
}
