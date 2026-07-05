import 'package:flutter/material.dart';

class DrawerMenu extends StatelessWidget {
  const DrawerMenu({super.key});

  @override
  Widget build(BuildContext context) {
    return Drawer(
      child: ListView(
        children: const [
          DrawerHeader(child: Text('HelixTerminator')),
          ListTile(leading: Icon(Icons.dashboard), title: Text('Dashboard')),
          ListTile(leading: Icon(Icons.computer), title: Text('Hosts')),
          ListTile(leading: Icon(Icons.terminal), title: Text('Terminal')),
          ListTile(leading: Icon(Icons.folder), title: Text('SFTP')),
          ListTile(leading: Icon(Icons.vpn_key), title: Text('Vault')),
          ListTile(leading: Icon(Icons.workspaces), title: Text('Workspaces')),
          ListTile(leading: Icon(Icons.settings), title: Text('Settings')),
        ],
      ),
    );
  }
}
