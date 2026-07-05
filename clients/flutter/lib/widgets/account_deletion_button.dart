import 'package:flutter/material.dart';

class AccountDeletionButton extends StatelessWidget {
  final VoidCallback? onDelete;

  const AccountDeletionButton({super.key, this.onDelete});

  @override
  Widget build(BuildContext context) {
    return ElevatedButton(
      style: ElevatedButton.styleFrom(backgroundColor: Colors.red),
      onPressed: onDelete,
      child: const Text('Delete Account'),
    );
  }
}
