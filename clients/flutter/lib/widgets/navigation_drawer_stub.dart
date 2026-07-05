import 'package:flutter/material.dart';

class NavigationDrawerStub extends StatelessWidget {
  final int selectedIndex;
  final ValueChanged<int>? onDestinationSelected;

  const NavigationDrawerStub({super.key, required this.selectedIndex, this.onDestinationSelected});

  @override
  Widget build(BuildContext context) {
    return NavigationDrawer(
      selectedIndex: selectedIndex,
      onDestinationSelected: onDestinationSelected,
      children: const [
        NavigationDrawerDestination(icon: Icon(Icons.dashboard), label: Text('Dashboard')),
        NavigationDrawerDestination(icon: Icon(Icons.computer), label: Text('Hosts')),
        NavigationDrawerDestination(icon: Icon(Icons.terminal), label: Text('Terminal')),
        NavigationDrawerDestination(icon: Icon(Icons.settings), label: Text('Settings')),
      ],
    );
  }
}
