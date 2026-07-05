import 'package:flutter/material.dart';

class DateRangePickerScale extends StatelessWidget {
  const DateRangePickerScale({super.key});

  @override
  Widget build(BuildContext context) {
    return ElevatedButton(
      onPressed: () => showDateRangePicker(
        context: context,
        firstDate: DateTime(2020),
        lastDate: DateTime(2100),
      ),
      child: const Text('Date Range Picker'),
    );
  }
}
