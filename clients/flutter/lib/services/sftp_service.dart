import '../models/sftp_file.dart';
import 'api_client.dart';

class SftpService {
  final ApiClient _apiClient;

  SftpService({required ApiClient apiClient}) : _apiClient = apiClient;

  Future<List<SftpFile>> listDirectory(String path) async {
    final response = await _apiClient.get('/api/v1/sftp/list?path=${Uri.encodeComponent(path)}');
    final data = response['data'] as List<dynamic>? ?? [];
    return data.map((json) => _sftpFileFromJson(json as Map<String, dynamic>)).toList();
  }

  Future<void> upload(String localPath, String remotePath) async {
    await _apiClient.post('/api/v1/sftp/upload', {
      'localPath': localPath,
      'remotePath': remotePath,
    });
  }

  Future<void> download(String remotePath, String localPath) async {
    await _apiClient.post('/api/v1/sftp/download', {
      'remotePath': remotePath,
      'localPath': localPath,
    });
  }

  Future<void> delete(String remotePath) async {
    await _apiClient.post('/api/v1/sftp/delete', {'path': remotePath});
  }

  Future<void> rename(String oldPath, String newPath) async {
    await _apiClient.post('/api/v1/sftp/rename', {
      'oldPath': oldPath,
      'newPath': newPath,
    });
  }

  Future<void> createDirectory(String remotePath) async {
    await _apiClient.post('/api/v1/sftp/mkdir', {'path': remotePath});
  }

  SftpFile _sftpFileFromJson(Map<String, dynamic> json) {
    return SftpFile(
      name: json['name'] as String,
      path: json['path'] as String,
      isDirectory: json['isDirectory'] as bool? ?? false,
      size: json['size'] as int? ?? 0,
      modifiedAt: DateTime.parse(json['modifiedAt'] as String),
    );
  }
}
