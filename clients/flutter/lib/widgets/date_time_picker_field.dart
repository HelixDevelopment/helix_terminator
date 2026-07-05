import 'package:flutter/material.dart';

class DateTimePickerField extends StatelessWidget {
  final DateTime? selected;
  final ValueChanged<DateTime>? onChanged;

  const DateTimePickerField({super.key, this.selected, this.onChanged});

  @override
  Widget build(BuildContext context) {
    return TextButton(
      onPressed: () async {
        final date = await showDatePicker(
          context: context,
          initialDate: selected ?? DateTime.now(),
          firstDate: DateTime(2020),
          lastDate: DateTime(2100),
        );
        if (date != null) onChanged?.call(date);
      },
      child: Text(selected?.toIso8601String() ?? 'Select date'),
    );
  }
}
