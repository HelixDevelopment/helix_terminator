import 'package:flutter/material.dart';

class ExpansionPanelListScale extends StatelessWidget {
  const ExpansionPanelListScale({super.key});

  @override
  Widget build(BuildContext context) {
    return ExpansionPanelList(
      expansionCallback: (panelIndex, isExpanded) {},
      children: [
        ExpansionPanel(
          headerBuilder: (context, isExpanded) => const ListTile(title: Text('Panel 1')),
          body: const ListTile(title: Text('Content 1')),
        ),
        ExpansionPanel(
          headerBuilder: (context, isExpanded) => const ListTile(title: Text('Panel 2')),
          body: const ListTile(title: Text('Content 2')),
        ),
      ],
    );
  }
}
