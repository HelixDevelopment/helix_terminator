import 'package:flutter/material.dart';

class DataTableScale extends StatelessWidget {
  const DataTableScale({super.key});

  @override
  Widget build(BuildContext context) {
    return DataTable(
      columns: const [
        DataColumn(label: Text('Name')),
        DataColumn(label: Text('Age')),
      ],
      rows: const [
        DataRow(cells: [DataCell(Text('Alice')), DataCell(Text('30'))]),
        DataRow(cells: [DataCell(Text('Bob')), DataCell(Text('25'))]),
      ],
    );
  }
}
