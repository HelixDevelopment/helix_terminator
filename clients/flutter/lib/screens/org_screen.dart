import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import '../bloc/org_bloc.dart';
import '../widgets/empty_state.dart';
import '../widgets/loading_indicator.dart';
import '../widgets/error_widget.dart' as helix_error;
import '../widgets/role_badge.dart';
import '../widgets/user_avatar.dart';

class OrgScreen extends StatefulWidget {
  const OrgScreen({super.key});

  @override
  State<OrgScreen> createState() => _OrgScreenState();
}

class _OrgScreenState extends State<OrgScreen> {
  @override
  void initState() {
    super.initState();
    context.read<OrgBloc>().add(OrgDashboardRequested());
  }

  void _showInviteDialog() {
    final emailController = TextEditingController();
    String selectedRole = 'viewer';

    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Invite Member'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              controller: emailController,
              decoration: const InputDecoration(labelText: 'Email'),
              keyboardType: TextInputType.emailAddress,
            ),
            const SizedBox(height: 16),
            DropdownButtonFormField<String>(
              value: selectedRole,
              decoration: const InputDecoration(labelText: 'Role'),
              items: const [
                DropdownMenuItem(value: 'admin', child: Text('Admin')),
                DropdownMenuItem(value: 'editor', child: Text('Editor')),
                DropdownMenuItem(value: 'viewer', child: Text('Viewer')),
              ],
              onChanged: (value) {
                if (value != null) selectedRole = value;
              },
            ),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context), child: const Text('Cancel')),
          FilledButton(
            onPressed: () {
              if (emailController.text.isNotEmpty) {
                context.read<OrgBloc>().add(OrgInviteMember(emailController.text, selectedRole));
                Navigator.pop(context);
              }
            },
            child: const Text('Invite'),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return DefaultTabController(
      length: 2,
      child: Scaffold(
        appBar: AppBar(
          title: const Text('Organization'),
          bottom: const TabBar(
            tabs: [
              Tab(icon: Icon(Icons.people), text: 'Members'),
              Tab(icon: Icon(Icons.settings), text: 'Settings'),
            ],
          ),
        ),
        body: BlocConsumer<OrgBloc, OrgState>(
          listener: (context, state) {
            if (state is OrgActionSuccess) {
              ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(state.message)));
            }
          },
          builder: (context, state) {
            if (state is OrgLoading) {
              return const LoadingIndicator();
            }
            if (state is OrgError) {
              return helix_error.ErrorWidget(
                message: state.message,
                onRetry: () => context.read<OrgBloc>().add(OrgDashboardRequested()),
              );
            }
            if (state is OrgDashboardLoaded) {
              return TabBarView(
                children: [
                  _MembersTab(
                    members: state.members,
                    onInvite: _showInviteDialog,
                  ),
                  _SettingsTab(organization: state.organization),
                ],
              );
            }
            return const SizedBox.shrink();
          },
        ),
        floatingActionButton: FloatingActionButton(
          onPressed: _showInviteDialog,
          child: const Icon(Icons.person_add),
        ),
      ),
    );
  }
}

class _MembersTab extends StatelessWidget {
  final List<Map<String, dynamic>> members;
  final VoidCallback onInvite;

  const _MembersTab({required this.members, required this.onInvite});

  @override
  Widget build(BuildContext context) {
    if (members.isEmpty) {
      return EmptyState(
        message: 'No members yet',
        action: FilledButton.icon(
          onPressed: onInvite,
          icon: const Icon(Icons.person_add),
          label: const Text('Invite Member'),
        ),
      );
    }

    return ListView.builder(
      itemCount: members.length,
      itemBuilder: (context, index) {
        final member = members[index];
        return Card(
          margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
          child: ListTile(
            leading: UserAvatar(
              imageUrl: member['avatarUrl'] as String?,
              initials: (member['name'] as String? ?? 'U').substring(0, 1).toUpperCase(),
            ),
            title: Text(member['name'] as String? ?? 'Unknown'),
            subtitle: Text(member['email'] as String? ?? ''),
            trailing: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                RoleBadge(role: member['role'] as String? ?? 'viewer'),
                PopupMenuButton<String>(
                  onSelected: (value) {
                    if (value == 'remove') {
                      context.read<OrgBloc>().add(OrgRemoveMember(member['id'] as String));
                    } else if (value.startsWith('role_')) {
                      final role = value.substring(5);
                      context.read<OrgBloc>().add(OrgUpdateMemberRole(member['id'] as String, role));
                    }
                  },
                  itemBuilder: (context) => [
                    const PopupMenuItem(value: 'role_admin', child: Text('Set as Admin')),
                    const PopupMenuItem(value: 'role_editor', child: Text('Set as Editor')),
                    const PopupMenuItem(value: 'role_viewer', child: Text('Set as Viewer')),
                    const PopupMenuItem(value: 'remove', child: Text('Remove')),
                  ],
                ),
              ],
            ),
          ),
        );
      },
    );
  }
}

class _SettingsTab extends StatefulWidget {
  final dynamic organization;

  const _SettingsTab({required this.organization});

  @override
  State<_SettingsTab> createState() => _SettingsTabState();
}

class _SettingsTabState extends State<_SettingsTab> {
  late final TextEditingController _nameController;
  late final TextEditingController _slugController;

  @override
  void initState() {
    super.initState();
    _nameController = TextEditingController(text: widget.organization?.name ?? '');
    _slugController = TextEditingController(text: widget.organization?.slug ?? '');
  }

  @override
  void dispose() {
    _nameController.dispose();
    _slugController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('Organization Details', style: Theme.of(context).textTheme.titleLarge),
          const SizedBox(height: 16),
          TextField(
            controller: _nameController,
            decoration: const InputDecoration(labelText: 'Organization Name'),
          ),
          const SizedBox(height: 16),
          TextField(
            controller: _slugController,
            decoration: const InputDecoration(labelText: 'Slug'),
          ),
          const SizedBox(height: 24),
          FilledButton.icon(
            onPressed: () {
              context.read<OrgBloc>().add(OrgUpdateSettings(
                name: _nameController.text,
                slug: _slugController.text,
              ));
            },
            icon: const Icon(Icons.save),
            label: const Text('Save Changes'),
          ),
          const SizedBox(height: 32),
          Text('Danger Zone', style: Theme.of(context).textTheme.titleLarge?.copyWith(color: Colors.red)),
          const SizedBox(height: 16),
          Card(
            color: Colors.red.shade50,
            child: ListTile(
              leading: const Icon(Icons.delete_forever, color: Colors.red),
              title: const Text('Delete Organization', style: TextStyle(color: Colors.red)),
              subtitle: const Text('This action cannot be undone'),
              onTap: () {
                showDialog(
                  context: context,
                  builder: (context) => AlertDialog(
                    title: const Text('Delete Organization?'),
                    content: const Text('This will permanently delete the organization and all associated data.'),
                    actions: [
                      TextButton(onPressed: () => Navigator.pop(context), child: const Text('Cancel')),
                      FilledButton(
                        style: FilledButton.styleFrom(backgroundColor: Colors.red),
                        onPressed: () {
                          Navigator.pop(context);
                          // TODO: implement delete
                        },
                        child: const Text('Delete'),
                      ),
                    ],
                  ),
                );
              },
            ),
          ),
        ],
      ),
    );
  }
}
