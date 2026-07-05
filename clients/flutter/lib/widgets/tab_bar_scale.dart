import 'package:flutter/material.dart';

class TabBarScale extends StatelessWidget {
  const TabBarScale({super.key});

  @override
  Widget build(BuildContext context) {
    return DefaultTabController(
      length: 3,
      child: Column(
        children: [
          const TabBar(tabs: [Tab(text: 'Tab 1'), Tab(text: 'Tab 2'), Tab(text: 'Tab 3')]),
          Expanded(
            child: TabBarView(
              children: [
                Container(color: Colors.red.shade100),
                Container(color: Colors.green.shade100),
                Container(color: Colors.blue.shade100),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
