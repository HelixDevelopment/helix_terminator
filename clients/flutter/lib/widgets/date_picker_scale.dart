import 'package:flutter/material.dart';

class DatePickerScale extends StatelessWidget {
  const DatePickerScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        ElevatedButton(
          onPressed: () => showDatePicker(
            context: context,
            initialDate: DateTime.now(),
            firstDate: DateTime(2020),
            lastDate: DateTime(2100),
          ),
          child: const Text('Date Picker'),
        ),
        ElevatedButton(
          onPressed: () => showTimePicker(
            context: context,
            initialTime: TimeOfDay.now(),
          ),
          child: const Text('Time Picker'),
        ),
      ],
    );
  }
}
