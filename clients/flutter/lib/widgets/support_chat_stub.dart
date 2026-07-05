import 'package:flutter/material.dart';

class SupportChatStub extends StatelessWidget {
  final VoidCallback? onSend;

  const SupportChatStub({super.key, this.onSend});

  @override
  Widget build(BuildContext context) {
    // TODO: integrate chat support widget
    return Container(
      height: 400,
      color: Colors.grey.shade200,
      child: const Center(child: Text('Support Chat')),
    );
  }
}
