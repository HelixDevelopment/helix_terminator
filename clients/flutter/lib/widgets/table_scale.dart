import 'package:flutter/material.dart';

class TableScale extends StatelessWidget {
  const TableScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Table(
      border: TableBorder.all(),
      children: const [
        TableRow(children: [Text('A1'), Text('B1')]),
        TableRow(children: [Text('A2'), Text('B2')]),
      ],
    );
  }
}
