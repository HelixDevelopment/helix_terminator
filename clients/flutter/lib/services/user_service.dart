import '../models/user.dart';
import 'api_client.dart';

class UserServiceException implements Exception {
  final String message;
  UserServiceException(this.message);
}

class UserService {
  final ApiClient _apiClient;

  UserService({required ApiClient apiClient}) : _apiClient = apiClient;

  Future<User> getCurrentUser() async {
    try {
      final response = await _apiClient.get('/api/v1/user/me');
      return _userFromJson(response['data'] as Map<String, dynamic>);
    } on ApiException catch (e) {
      throw UserServiceException(e.message);
    } catch (e) {
      throw UserServiceException('Failed to load user');
    }
  }

  Future<User> updateProfile({
    String? name,
    String? avatarUrl,
  }) async {
    try {
      final body = <String, dynamic>{};
      if (name != null) body['name'] = name;
      if (avatarUrl != null) body['avatarUrl'] = avatarUrl;
      final response = await _apiClient.post('/api/v1/user/me', body);
      return _userFromJson(response['data'] as Map<String, dynamic>);
    } on ApiException catch (e) {
      throw UserServiceException(e.message);
    } catch (e) {
      throw UserServiceException('Failed to update profile');
    }
  }

  Future<void> changePassword(String currentPassword, String newPassword) async {
    try {
      await _apiClient.post('/api/v1/user/change-password', {
        'currentPassword': currentPassword,
        'newPassword': newPassword,
      });
    } on ApiException catch (e) {
      throw UserServiceException(e.message);
    } catch (e) {
      throw UserServiceException('Failed to change password');
    }
  }

  Future<void> enableTwoFactor() async {
    try {
      await _apiClient.post('/api/v1/user/2fa/enable', {});
    } on ApiException catch (e) {
      throw UserServiceException(e.message);
    } catch (e) {
      throw UserServiceException('Failed to enable 2FA');
    }
  }

  Future<void> disableTwoFactor(String code) async {
    try {
      await _apiClient.post('/api/v1/user/2fa/disable', {'code': code});
    } on ApiException catch (e) {
      throw UserServiceException(e.message);
    } catch (e) {
      throw UserServiceException('Failed to disable 2FA');
    }
  }

  Future<Map<String, dynamic>> getTwoFactorSetup() async {
    try {
      final response = await _apiClient.get('/api/v1/user/2fa/setup');
      return response['data'] as Map<String, dynamic>? ?? {};
    } on ApiException catch (e) {
      throw UserServiceException(e.message);
    } catch (e) {
      throw UserServiceException('Failed to load 2FA setup');
    }
  }

  Future<void> updatePreferences(Map<String, dynamic> preferences) async {
    try {
      await _apiClient.post('/api/v1/user/preferences', preferences);
    } on ApiException catch (e) {
      throw UserServiceException(e.message);
    } catch (e) {
      throw UserServiceException('Failed to update preferences');
    }
  }

  Future<Map<String, dynamic>> getPreferences() async {
    try {
      final response = await _apiClient.get('/api/v1/user/preferences');
      return response['data'] as Map<String, dynamic>? ?? {};
    } on ApiException catch (e) {
      throw UserServiceException(e.message);
    } catch (e) {
      throw UserServiceException('Failed to load preferences');
    }
  }

  User _userFromJson(Map<String, dynamic> json) {
    return User(
      id: json['id'] as String,
      email: json['email'] as String,
      name: json['name'] as String,
      avatarUrl: json['avatarUrl'] as String?,
      createdAt: DateTime.parse(json['createdAt'] as String),
    );
  }
}
