import 'package:flutter/material.dart';

class InfiniteScrollList extends StatelessWidget {
  final List<Widget> children;
  final VoidCallback? onLoadMore;

  const InfiniteScrollList({super.key, required this.children, this.onLoadMore});

  @override
  Widget build(BuildContext context) {
    return NotificationListener<ScrollNotification>(
      onNotification: (notification) {
        if (notification.metrics.pixels >= notification.metrics.maxScrollExtent - 200) {
          onLoadMore?.call();
        }
        return false;
      },
      child: ListView(children: children),
    );
  }
}
