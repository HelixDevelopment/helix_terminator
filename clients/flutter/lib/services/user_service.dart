import '../models/user.dart';
import 'api_client.dart';

class UserService {
  final ApiClient _apiClient;

  UserService({required ApiClient apiClient}) : _apiClient = apiClient;

  Future<User> getCurrentUser() async {
    final response = await _apiClient.get('/api/v1/user/me');
    return _userFromJson(response['data'] as Map<String, dynamic>);
  }

  Future<User> updateProfile({
    String? name,
    String? avatarUrl,
  }) async {
    final body = <String, dynamic>{};
    if (name != null) body['name'] = name;
    if (avatarUrl != null) body['avatarUrl'] = avatarUrl;
    final response = await _apiClient.post('/api/v1/user/me', body);
    return _userFromJson(response['data'] as Map<String, dynamic>);
  }

  Future<void> changePassword(String currentPassword, String newPassword) async {
    await _apiClient.post('/api/v1/user/change-password', {
      'currentPassword': currentPassword,
      'newPassword': newPassword,
    });
  }

  Future<void> enableTwoFactor() async {
    await _apiClient.post('/api/v1/user/2fa/enable', {});
  }

  Future<void> disableTwoFactor(String code) async {
    await _apiClient.post('/api/v1/user/2fa/disable', {'code': code});
  }

  Future<Map<String, dynamic>> getTwoFactorSetup() async {
    final response = await _apiClient.get('/api/v1/user/2fa/setup');
    return response['data'] as Map<String, dynamic>? ?? {};
  }

  Future<void> updatePreferences(Map<String, dynamic> preferences) async {
    await _apiClient.post('/api/v1/user/preferences', preferences);
  }

  Future<Map<String, dynamic>> getPreferences() async {
    final response = await _apiClient.get('/api/v1/user/preferences');
    return response['data'] as Map<String, dynamic>? ?? {};
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
