import 'package:flutter/material.dart';

class HostConnectionDialog extends StatelessWidget {
  final String hostName;
  final VoidCallback? onConnect;

  const HostConnectionDialog({super.key, required this.hostName, this.onConnect});

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: Text('Connect to $hostName'),
      content: const Text('Choose connection method:'),
      actions: [
        TextButton(onPressed: () {}, child: const Text('SSH')),
        TextButton(onPressed: () {}, child: const Text('SFTP')),
        ElevatedButton(onPressed: onConnect, child: const Text('Connect')),
      ],
    );
  }
}
