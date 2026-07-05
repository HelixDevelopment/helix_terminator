import 'package:flutter/material.dart';

class InlineSearch extends StatelessWidget {
  final TextEditingController? controller;
  final ValueChanged<String>? onChanged;

  const InlineSearch({super.key, this.controller, this.onChanged});

  @override
  Widget build(BuildContext context) {
    return TextField(
      controller: controller,
      decoration: const InputDecoration(
        hintText: 'Search...',
        prefixIcon: Icon(Icons.search),
        border: InputBorder.none,
      ),
      onChanged: onChanged,
    );
  }
}
