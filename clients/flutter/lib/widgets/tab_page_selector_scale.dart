import 'package:flutter/material.dart';

class TabPageSelectorScale extends StatelessWidget {
  const TabPageSelectorScale({super.key});

  @override
  Widget build(BuildContext context) {
    return DefaultTabController(
      length: 3,
      child: Column(
        children: [
          const TabPageSelector(),
          Expanded(
            child: TabBarView(
              children: [
                Container(color: Colors.red),
                Container(color: Colors.green),
                Container(color: Colors.blue),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
