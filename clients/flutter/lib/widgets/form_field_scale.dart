import 'package:flutter/material.dart';

class FormFieldScale extends StatelessWidget {
  const FormFieldScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        TextField(decoration: const InputDecoration(labelText: 'Text')),
        const SizedBox(height: 8),
        TextField(decoration: const InputDecoration(labelText: 'Password'), obscureText: true),
        const SizedBox(height: 8),
        DropdownButtonFormField<String>(
          items: const [DropdownMenuItem(value: 'a', child: Text('A'))],
          onChanged: (_) {},
          decoration: const InputDecoration(labelText: 'Dropdown'),
        ),
        const SizedBox(height: 8),
        CheckboxListTile(value: false, onChanged: (_) {}, title: const Text('Checkbox')),
        SwitchListTile(value: false, onChanged: (_) {}, title: const Text('Switch')),
        RadioListTile(value: 'a', groupValue: 'a', onChanged: (_) {}, title: const Text('Radio')),
      ],
    );
  }
}
