import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import '../bloc/host_bloc.dart';
import '../models/host.dart';

class HostCreateScreen extends StatefulWidget {
  final Host? host;

  const HostCreateScreen({super.key, this.host});

  @override
  State<HostCreateScreen> createState() => _HostCreateScreenState();
}

class _HostCreateScreenState extends State<HostCreateScreen> {
  final _formKey = GlobalKey<FormState>();
  late final TextEditingController _nameController;
  late final TextEditingController _hostnameController;
  late final TextEditingController _portController;
  late final TextEditingController _usernameController;
  late final TextEditingController _passwordController;
  late final TextEditingController _tagsController;

  String _authMethod = 'password';
  String? _selectedOrganization;
  bool _isSubmitting = false;

  final List<String> _organizations = ['Personal', 'Work', 'Staging', 'Production'];

  @override
  void initState() {
    super.initState();
    final host = widget.host;
    _nameController = TextEditingController(text: host?.name ?? '');
    _hostnameController = TextEditingController(text: host?.address ?? '');
    _portController = TextEditingController(text: host?.port.toString() ?? '22');
    _usernameController = TextEditingController(text: host?.username ?? '');
    _passwordController = TextEditingController();
    _tagsController = TextEditingController(text: host?.tags.join(', ') ?? '');
    if (host != null) {
      _authMethod = host.authMethod;
      _selectedOrganization = host.organizationId;
    }
  }

  @override
  void dispose() {
    _nameController.dispose();
    _hostnameController.dispose();
    _portController.dispose();
    _usernameController.dispose();
    _passwordController.dispose();
    _tagsController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isEditing = widget.host != null;
    final isDesktop = MediaQuery.of(context).size.width > 800;

    return Scaffold(
      appBar: AppBar(
        title: Text(isEditing ? 'Edit Host' : 'Add Host'),
      ),
      body: BlocListener<HostBloc, HostState>(
        listener: (context, state) {
          if (state is HostOperationSuccess) {
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(content: Text(state.message)),
            );
            Navigator.of(context).pop();
          } else if (state is HostError) {
            setState(() => _isSubmitting = false);
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(content: Text(state.message), backgroundColor: theme.colorScheme.error),
            );
          }
        },
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(16.0),
          child: Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 700),
              child: Form(
                key: _formKey,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Card(
                      elevation: 2,
                      child: Padding(
                        padding: const EdgeInsets.all(20.0),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text('Basic Information', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
                            const SizedBox(height: 16),
                            TextFormField(
                              controller: _nameController,
                              decoration: const InputDecoration(
                                labelText: 'Name',
                                hintText: 'My Production Server',
                                prefixIcon: Icon(Icons.label),
                              ),
                              validator: (value) {
                                if (value == null || value.trim().isEmpty) {
                                  return 'Name is required';
                                }
                                return null;
                              },
                              textInputAction: TextInputAction.next,
                            ),
                            const SizedBox(height: 16),
                            TextFormField(
                              controller: _hostnameController,
                              decoration: const InputDecoration(
                                labelText: 'Hostname / IP Address',
                                hintText: '192.168.1.100 or server.example.com',
                                prefixIcon: Icon(Icons.computer),
                              ),
                              validator: (value) {
                                if (value == null || value.trim().isEmpty) {
                                  return 'Hostname is required';
                                }
                                return null;
                              },
                              textInputAction: TextInputAction.next,
                            ),
                            const SizedBox(height: 16),
                            TextFormField(
                              controller: _portController,
                              decoration: const InputDecoration(
                                labelText: 'Port',
                                prefixIcon: Icon(Icons.numbers),
                              ),
                              keyboardType: TextInputType.number,
                              validator: (value) {
                                if (value == null || value.trim().isEmpty) {
                                  return 'Port is required';
                                }
                                final port = int.tryParse(value);
                                if (port == null || port < 1 || port > 65535) {
                                  return 'Enter a valid port (1-65535)';
                                }
                                return null;
                              },
                              textInputAction: TextInputAction.next,
                            ),
                          ],
                        ),
                      ),
                    ),
                    const SizedBox(height: 16),
                    Card(
                      elevation: 2,
                      child: Padding(
                        padding: const EdgeInsets.all(20.0),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text('Authentication', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
                            const SizedBox(height: 16),
                            TextFormField(
                              controller: _usernameController,
                              decoration: const InputDecoration(
                                labelText: 'Username',
                                hintText: 'root',
                                prefixIcon: Icon(Icons.person),
                              ),
                              textInputAction: TextInputAction.next,
                            ),
                            const SizedBox(height: 16),
                            SegmentedButton<String>(
                              segments: const [
                                ButtonSegment(value: 'password', label: Text('Password'), icon: Icon(Icons.password)),
                                ButtonSegment(value: 'ssh_key', label: Text('SSH Key'), icon: Icon(Icons.key)),
                              ],
                              selected: {_authMethod},
                              onSelectionChanged: (set) => setState(() => _authMethod = set.first),
                            ),
                            const SizedBox(height: 16),
                            if (_authMethod == 'password')
                              TextFormField(
                                controller: _passwordController,
                                decoration: const InputDecoration(
                                  labelText: 'Password',
                                  prefixIcon: Icon(Icons.lock),
                                ),
                                obscureText: true,
                                textInputAction: TextInputAction.next,
                              )
                            else
                              TextFormField(
                                controller: _passwordController,
                                decoration: const InputDecoration(
                                  labelText: 'Private Key',
                                  hintText: 'Paste your private SSH key here',
                                  prefixIcon: Icon(Icons.key),
                                ),
                                maxLines: 5,
                                textInputAction: TextInputAction.next,
                              ),
                          ],
                        ),
                      ),
                    ),
                    const SizedBox(height: 16),
                    Card(
                      elevation: 2,
                      child: Padding(
                        padding: const EdgeInsets.all(20.0),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text('Organization & Tags', style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
                            const SizedBox(height: 16),
                            DropdownButtonFormField<String>(
                              value: _selectedOrganization,
                              decoration: const InputDecoration(
                                labelText: 'Organization',
                                prefixIcon: Icon(Icons.business),
                              ),
                              items: _organizations
                                  .map((org) => DropdownMenuItem(value: org.toLowerCase(), child: Text(org)))
                                  .toList(),
                              onChanged: (value) => setState(() => _selectedOrganization = value),
                              hint: const Text('Select organization'),
                            ),
                            const SizedBox(height: 16),
                            TextFormField(
                              controller: _tagsController,
                              decoration: const InputDecoration(
                                labelText: 'Tags',
                                hintText: 'web, production, aws (comma separated)',
                                prefixIcon: Icon(Icons.tag),
                              ),
                              textInputAction: TextInputAction.done,
                            ),
                          ],
                        ),
                      ),
                    ),
                    const SizedBox(height: 24),
                    SizedBox(
                      width: double.infinity,
                      height: 52,
                      child: FilledButton.icon(
                        onPressed: _isSubmitting ? null : _submit,
                        icon: _isSubmitting
                            ? const SizedBox(
                                width: 20,
                                height: 20,
                                child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                              )
                            : Icon(isEditing ? Icons.save : Icons.add),
                        label: Text(isEditing ? 'Save Changes' : 'Create Host'),
                      ),
                    ),
                    const SizedBox(height: 32),
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }

  void _submit() {
    if (!_formKey.currentState!.validate()) return;

    setState(() => _isSubmitting = true);

    final port = int.parse(_portController.text.trim());
    final tags = _tagsController.text
        .split(',')
        .map((t) => t.trim())
        .where((t) => t.isNotEmpty)
        .toList();

    final host = Host(
      id: widget.host?.id ?? DateTime.now().millisecondsSinceEpoch.toString(),
      name: _nameController.text.trim(),
      address: _hostnameController.text.trim(),
      port: port,
      username: _usernameController.text.trim().isEmpty ? null : _usernameController.text.trim(),
      tags: tags,
      createdAt: widget.host?.createdAt ?? DateTime.now(),
      status: widget.host?.status ?? 'unknown',
      organizationId: _selectedOrganization,
      authMethod: _authMethod,
    );

    if (widget.host != null) {
      context.read<HostBloc>().add(HostUpdateRequested(widget.host!.id, host));
    } else {
      context.read<HostBloc>().add(HostCreateRequested(host));
    }
  }
}
