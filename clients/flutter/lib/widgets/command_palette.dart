import 'package:flutter/material.dart';

class CommandPalette extends StatelessWidget {
  final List<String> commands;
  final ValueChanged<String>? onCommandSelected;

  const CommandPalette({super.key, required this.commands, this.onCommandSelected});

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('Command Palette'),
      content: SizedBox(
        width: 400,
        child: ListView.builder(
          shrinkWrap: true,
          itemCount: commands.length,
          itemBuilder: (context, index) {
            return ListTile(
              title: Text(commands[index]),
              onTap: () => onCommandSelected?.call(commands[index]),
            );
          },
        ),
      ),
    );
  }
}
