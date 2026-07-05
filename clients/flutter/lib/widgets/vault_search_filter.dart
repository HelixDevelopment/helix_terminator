import 'package:flutter/material.dart';

class VaultSearchFilter extends StatelessWidget {
  final ValueChanged<String>? onSearch;
  final ValueChanged<String>? onFilterType;

  const VaultSearchFilter({super.key, this.onSearch, this.onFilterType});

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(
          child: TextField(
            decoration: const InputDecoration(hintText: 'Search secrets...'),
            onChanged: onSearch,
          ),
        ),
        DropdownButton<String>(
          hint: const Text('Type'),
          items: const [
            DropdownMenuItem(value: 'password', child: Text('Password')),
            DropdownMenuItem(value: 'key', child: Text('SSH Key')),
            DropdownMenuItem(value: 'token', child: Text('Token')),
          ],
          onChanged: (v) => onFilterType?.call(v!),
        ),
      ],
    );
  }
}
