import 'package:flutter/material.dart';

class InviteUserDialog extends StatelessWidget {
  final TextEditingController? emailController;
  final VoidCallback? onInvite;

  const InviteUserDialog({super.key, this.emailController, this.onInvite});

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('Invite User'),
      content: TextField(
        controller: emailController,
        decoration: const InputDecoration(hintText: 'Email address'),
        keyboardType: TextInputType.emailAddress,
      ),
      actions: [
        TextButton(onPressed: () => Navigator.of(context).pop(), child: const Text('Cancel')),
        ElevatedButton(onPressed: onInvite, child: const Text('Invite')),
      ],
    );
  }
}
