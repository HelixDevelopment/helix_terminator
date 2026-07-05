import 'package:flutter/material.dart';

class AdaptiveScaffold extends StatelessWidget {
  final Widget body;
  final Widget? sidebar;
  final PreferredSizeWidget? appBar;
  final Widget? bottomNav;
  final Widget? floatingActionButton;

  const AdaptiveScaffold({
    super.key,
    required this.body,
    this.sidebar,
    this.appBar,
    this.bottomNav,
    this.floatingActionButton,
  });

  @override
  Widget build(BuildContext context) {
    final width = MediaQuery.of(context).size.width;
    return Scaffold(
      appBar: appBar,
      bottomNavigationBar: bottomNav,
      floatingActionButton: floatingActionButton,
      body: Row(
        children: [
          if (width >= 1200 && sidebar != null) sidebar!,
          Expanded(child: body),
        ],
      ),
    );
  }
}
