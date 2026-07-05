import 'package:flutter/material.dart';

class FeedbackForm extends StatelessWidget {
  final TextEditingController? controller;
  final VoidCallback? onSubmit;

  const FeedbackForm({super.key, this.controller, this.onSubmit});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        TextField(
          controller: controller,
          maxLines: 5,
          decoration: const InputDecoration(hintText: 'Your feedback...'),
        ),
        ElevatedButton(onPressed: onSubmit, child: const Text('Submit')),
      ],
    );
  }
}
