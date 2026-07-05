import 'package:flutter/material.dart';

class PaginatedDataTableScale extends StatelessWidget {
  const PaginatedDataTableScale({super.key});

  @override
  Widget build(BuildContext context) {
    return PaginatedDataTable(
      header: const Text('Data'),
      rowsPerPage: 5,
      columns: const [
        DataColumn(label: Text('Name')),
        DataColumn(label: Text('Age')),
      ],
      source: _DataSource(),
    );
  }
}

class _DataSource extends DataTableSource {
  @override
  DataRow? getRow(int index) {
    return DataRow(cells: [DataCell(Text('Name $index')), DataCell(Text('$index'))]);
  }

  @override
  bool get isRowCountApproximate => false;

  @override
  int get rowCount => 20;

  @override
  int get selectedRowCount => 0;
}
