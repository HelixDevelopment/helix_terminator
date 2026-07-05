import 'package:flutter/material.dart';

class AnimationDurationScale extends StatelessWidget {
  const AnimationDurationScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [100, 200, 300, 500, 1000].map((ms) {
        return ListTile(
          title: Text('$ms ms'),
          trailing: ElevatedButton(
            onPressed: () {},
            child: const Text('Test'),
          ),
        );
      }).toList(),
    );
  }
}
