import 'package:flutter/material.dart';

class RefreshIndicatorScale extends StatelessWidget {
  const RefreshIndicatorScale({super.key});

  @override
  Widget build(BuildContext context) {
    return RefreshIndicator(
      onRefresh: () async {},
      child: ListView.builder(
        itemCount: 10,
        itemBuilder: (context, index) => ListTile(title: Text('Item $index')),
      ),
    );
  }
}
