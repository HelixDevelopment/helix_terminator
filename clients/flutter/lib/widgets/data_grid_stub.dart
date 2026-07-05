import 'package:flutter/material.dart';

class DataGridStub extends StatelessWidget {
  final List<DataColumn> columns;
  final List<DataRow> rows;

  const DataGridStub({super.key, required this.columns, required this.rows});

  @override
  Widget build(BuildContext context) {
    // TODO: integrate syncfusion_flutter_datagrid or pluto_grid
    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      child: DataTable(columns: columns, rows: rows),
    );
  }
}
