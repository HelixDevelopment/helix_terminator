import 'package:flutter/material.dart';

class NavigationRailStub extends StatelessWidget {
  final int selectedIndex;
  final ValueChanged<int>? onDestinationSelected;

  const NavigationRailStub({super.key, required this.selectedIndex, this.onDestinationSelected});

  @override
  Widget build(BuildContext context) {
    return NavigationRail(
      selectedIndex: selectedIndex,
      onDestinationSelected: onDestinationSelected,
      destinations: const [
        NavigationRailDestination(icon: Icon(Icons.dashboard), label: Text('Dashboard')),
        NavigationRailDestination(icon: Icon(Icons.computer), label: Text('Hosts')),
        NavigationRailDestination(icon: Icon(Icons.terminal), label: Text('Terminal')),
        NavigationRailDestination(icon: Icon(Icons.settings), label: Text('Settings')),
      ],
    );
  }
}
