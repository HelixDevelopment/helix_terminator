import 'package:flutter/material.dart';

class PermissionMatrix extends StatelessWidget {
  final List<String> users;
  final List<String> permissions;
  final Map<String, Map<String, bool>> matrix;

  const PermissionMatrix({super.key, required this.users, required this.permissions, required this.matrix});

  @override
  Widget build(BuildContext context) {
    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      child: DataTable(
        columns: [
          const DataColumn(label: Text('User')),
          ...permissions.map((p) => DataColumn(label: Text(p))),
        ],
        rows: users.map((u) {
          return DataRow(
            cells: [
              DataCell(Text(u)),
              ...permissions.map((p) => DataCell(
                Checkbox(
                  value: matrix[u]?[p] ?? false,
                  onChanged: (_) {},
                ),
              )),
            ],
          );
        }).toList(),
      ),
    );
  }
}
