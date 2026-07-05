import 'package:flutter/material.dart';

class PinCodeField extends StatelessWidget {
  final int length;
  final ValueChanged<String>? onCompleted;

  const PinCodeField({super.key, this.length = 6, this.onCompleted});

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.center,
      children: List.generate(length, (index) {
        return Container(
          width: 40,
          height: 50,
          margin: const EdgeInsets.symmetric(horizontal: 4),
          decoration: BoxDecoration(
            border: Border.all(color: Colors.grey),
            borderRadius: BorderRadius.circular(8),
          ),
          child: const Center(child: Text('')),
        );
      }),
    );
  }
}
