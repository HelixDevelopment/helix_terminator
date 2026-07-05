import 'package:flutter/material.dart';

class SnackbarScale extends StatelessWidget {
  const SnackbarScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        ElevatedButton(
          onPressed: () => ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('Default')),
          ),
          child: const Text('Default Snackbar'),
        ),
        ElevatedButton(
          onPressed: () => ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(
              content: const Text('Action'),
              action: SnackBarAction(label: 'Undo', onPressed: () {}),
            ),
          ),
          child: const Text('Action Snackbar'),
        ),
      ],
    );
  }
}
