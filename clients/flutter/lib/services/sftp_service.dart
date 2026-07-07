import '../models/sftp_file.dart';
import 'api_client.dart';

class SftpServiceException implements Exception {
  final String message;
  SftpServiceException(this.message);
}

class SftpService {
  final ApiClient _apiClient;

  SftpService({required ApiClient apiClient}) : _apiClient = apiClient;

  Future<List<SftpFile>> listDirectory(String path) async {
    try {
      final response = await _apiClient.get('/api/v1/sftp/list?path=${Uri.encodeComponent(path)}');
      final data = response['data'] as List<dynamic>? ?? [];
      return data.map((json) => _sftpFileFromJson(json as Map<String, dynamic>)).toList();
    } on ApiException catch (e) {
      throw SftpServiceException(e.message);
    } catch (e) {
      throw SftpServiceException('Failed to list directory');
    }
  }

  Future<void> upload(String localPath, String remotePath) async {
    try {
      await _apiClient.post('/api/v1/sftp/upload', {
        'localPath': localPath,
        'remotePath': remotePath,
      });
    } on ApiException catch (e) {
      throw SftpServiceException(e.message);
    } catch (e) {
      throw SftpServiceException('Failed to upload file');
    }
  }

  Future<void> download(String remotePath, String localPath) async {
    try {
      await _apiClient.post('/api/v1/sftp/download', {
        'remotePath': remotePath,
        'localPath': localPath,
      });
    } on ApiException catch (e) {
      throw SftpServiceException(e.message);
    } catch (e) {
      throw SftpServiceException('Failed to download file');
    }
  }

  Future<void> delete(String remotePath) async {
    try {
      await _apiClient.post('/api/v1/sftp/delete', {'path': remotePath});
    } on ApiException catch (e) {
      throw SftpServiceException(e.message);
    } catch (e) {
      throw SftpServiceException('Failed to delete file');
    }
  }

  Future<void> rename(String oldPath, String newPath) async {
    try {
      await _apiClient.post('/api/v1/sftp/rename', {
        'oldPath': oldPath,
        'newPath': newPath,
      });
    } on ApiException catch (e) {
      throw SftpServiceException(e.message);
    } catch (e) {
      throw SftpServiceException('Failed to rename file');
    }
  }

  Future<void> createDirectory(String remotePath) async {
    try {
      await _apiClient.post('/api/v1/sftp/mkdir', {'path': remotePath});
    } on ApiException catch (e) {
      throw SftpServiceException(e.message);
    } catch (e) {
      throw SftpServiceException('Failed to create directory');
    }
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
