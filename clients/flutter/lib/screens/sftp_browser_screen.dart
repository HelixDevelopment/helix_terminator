import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import '../bloc/sftp_bloc.dart';
import '../widgets/empty_state.dart';
import '../widgets/loading_indicator.dart';
import '../widgets/error_widget.dart' as helix_error;
import '../widgets/breadcrumb_nav.dart';
import '../widgets/file_icon.dart';
import '../widgets/file_size_label.dart';

class SftpBrowserScreen extends StatefulWidget {
  final String? initialPath;
  const SftpBrowserScreen({super.key, this.initialPath});

  @override
  State<SftpBrowserScreen> createState() => _SftpBrowserScreenState();
}

class _SftpBrowserScreenState extends State<SftpBrowserScreen> {
  String _currentPath = '/';

  @override
  void initState() {
    super.initState();
    _currentPath = widget.initialPath ?? '/';
    context.read<SftpBloc>().add(SftpListDirectory(_currentPath));
  }

  void _navigateTo(String path) {
    setState(() => _currentPath = path);
    context.read<SftpBloc>().add(SftpListDirectory(path));
  }

  void _navigateUp() {
    if (_currentPath == '/') return;
    final segments = _currentPath.split('/').where((s) => s.isNotEmpty).toList();
    segments.removeLast();
    final newPath = segments.isEmpty ? '/' : '/${segments.join('/')}';
    _navigateTo(newPath);
  }

  void _showUploadDialog() {
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Upload File'),
        content: const Text('Select a file to upload to the current directory.'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context), child: const Text('Cancel')),
          FilledButton(
            onPressed: () {
              context.read<SftpBloc>().add(SftpUploadFile('/local/path', '$_currentPath/remote'));
              Navigator.pop(context);
            },
            child: const Text('Upload'),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final pathSegments = _currentPath.split('/').where((s) => s.isNotEmpty).toList();

    return Scaffold(
      appBar: AppBar(
        title: const Text('SFTP Browser'),
        actions: [
          IconButton(
            icon: const Icon(Icons.upload_file),
            tooltip: 'Upload',
            onPressed: _showUploadDialog,
          ),
        ],
      ),
      body: Column(
        children: [
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
            color: Theme.of(context).colorScheme.surfaceContainerHighest,
            child: Row(
              children: [
                IconButton(
                  icon: const Icon(Icons.arrow_upward),
                  tooltip: 'Parent directory',
                  onPressed: _navigateUp,
                ),
                Expanded(
                  child: BreadcrumbNav(
                    segments: ['Home', ...pathSegments],
                    onTapSegment: (index) {
                      if (index == 0) {
                        _navigateTo('/');
                      } else {
                        final newPath = '/${pathSegments.take(index).join('/')}';
                        _navigateTo(newPath);
                      }
                    },
                  ),
                ),
              ],
            ),
          ),
          Expanded(
            child: BlocConsumer<SftpBloc, SftpState>(
              listener: (context, state) {
                if (state is SftpActionSuccess) {
                  ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(state.message)));
                  context.read<SftpBloc>().add(SftpListDirectory(_currentPath));
                }
              },
              builder: (context, state) {
                if (state is SftpLoading) {
                  return const LoadingIndicator();
                }
                if (state is SftpError) {
                  return helix_error.ErrorWidget(
                    message: state.message,
                    onRetry: () => context.read<SftpBloc>().add(SftpListDirectory(_currentPath)),
                  );
                }
                if (state is SftpDirectoryLoaded) {
                  if (state.files.isEmpty) {
                    return const EmptyState(message: 'This directory is empty');
                  }
                  return ListView.builder(
                    itemCount: state.files.length,
                    itemBuilder: (context, index) {
                      final file = state.files[index];
                      return ListTile(
                        leading: Icon(
                          file.isDirectory ? Icons.folder : Icons.insert_drive_file,
                          color: file.isDirectory
                              ? Theme.of(context).colorScheme.primary
                              : Theme.of(context).colorScheme.onSurface,
                        ),
                        title: Text(file.name),
                        subtitle: Row(
                          children: [
                            FileSizeLabel(bytes: file.size),
                            const SizedBox(width: 12),
                            Text(
                              '${file.modifiedAt.day}/${file.modifiedAt.month}/${file.modifiedAt.year}',
                              style: Theme.of(context).textTheme.bodySmall,
                            ),
                          ],
                        ),
                        trailing: file.isDirectory
                            ? const Icon(Icons.chevron_right)
                            : Row(
                                mainAxisSize: MainAxisSize.min,
                                children: [
                                  IconButton(
                                    icon: const Icon(Icons.download),
                                    tooltip: 'Download',
                                    onPressed: () {
                                      context.read<SftpBloc>().add(
                                        SftpDownloadFile(file.path, '/local/${file.name}'),
                                      );
                                    },
                                  ),
                                  PopupMenuButton<String>(
                                    onSelected: (value) {
                                      if (value == 'delete') {
                                        context.read<SftpBloc>().add(SftpDeleteFile(file.path));
                                      } else if (value == 'rename') {
                                        _showRenameDialog(file.path);
                                      }
                                    },
                                    itemBuilder: (context) => [
                                      const PopupMenuItem(value: 'rename', child: Text('Rename')),
                                      const PopupMenuItem(value: 'delete', child: Text('Delete')),
                                    ],
                                  ),
                                ],
                              ),
                        onTap: () {
                          if (file.isDirectory) {
                            _navigateTo(file.path);
                          }
                        },
                      );
                    },
                  );
                }
                return const SizedBox.shrink();
              },
            ),
          ),
        ],
      ),
      floatingActionButton: FloatingActionButton(
        onPressed: _showUploadDialog,
        child: const Icon(Icons.upload_file),
      ),
    );
  }

  void _showRenameDialog(String path) {
    final nameController = TextEditingController();
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Rename'),
        content: TextField(
          controller: nameController,
          decoration: const InputDecoration(labelText: 'New name'),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context), child: const Text('Cancel')),
          FilledButton(
            onPressed: () {
              final newPath = path.substring(0, path.lastIndexOf('/') + 1) + nameController.text;
              context.read<SftpBloc>().add(SftpRenameFile(path, newPath));
              Navigator.pop(context);
            },
            child: const Text('Rename'),
          ),
        ],
      ),
    );
  }
}
