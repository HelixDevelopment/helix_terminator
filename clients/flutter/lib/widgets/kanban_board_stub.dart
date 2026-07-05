import 'package:flutter/material.dart';

class KanbanBoardStub extends StatelessWidget {
  final List<dynamic> columns;

  const KanbanBoardStub({super.key, required this.columns});

  @override
  Widget build(BuildContext context) {
    // TODO: integrate flutter_board or custom kanban
    return Container(
      height: 400,
      color: Colors.grey.shade200,
      child: const Center(child: Text('Kanban Board')),
    );
  }
}
