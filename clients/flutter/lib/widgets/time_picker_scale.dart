import 'package:flutter/material.dart';

class TimePickerScale extends StatelessWidget {
  const TimePickerScale({super.key});

  @override
  Widget build(BuildContext context) {
    return ElevatedButton(
      onPressed: () => showTimePicker(
        context: context,
        initialTime: TimeOfDay.now(),
      ),
      child: const Text('Time Picker'),
    );
  }
}
