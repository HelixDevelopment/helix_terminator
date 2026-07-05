import 'package:flutter/material.dart';

class TimeRangePicker extends StatelessWidget {
  final DateTime start;
  final DateTime end;
  final ValueChanged<DateTimeRange>? onChanged;

  const TimeRangePicker({super.key, required this.start, required this.end, this.onChanged});

  @override
  Widget build(BuildContext context) {
    return ElevatedButton(
      onPressed: () async {
        final range = await showDateRangePicker(
          context: context,
          firstDate: DateTime(2020),
          lastDate: DateTime.now(),
        );
        if (range != null) onChanged?.call(range);
      },
      child: Text('${start.toIso8601String()} - ${end.toIso8601String()}'),
    );
  }
}
