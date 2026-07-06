import 'dart:convert';
import 'package:shared_preferences/shared_preferences.dart';
import '../services/api_client.dart';
import '../models/user.dart';

class AuthException implements Exception {
  final String message;
  AuthException(this.message);
}

class AuthResult {
  final User? user;
  final bool requires2FA;
  final String? tempToken;

  AuthResult({
    this.user,
    this.requires2FA = false,
    this.tempToken,
  });
}

class AuthService {
  final ApiClient _apiClient;
  static const _tokenKey = 'auth_token';
  static const _refreshTokenKey = 'refresh_token';

  AuthService({required ApiClient apiClient}) : _apiClient = apiClient;

  Future<AuthResult> login(String email, String password) async {
    try {
      final response = await _apiClient.post('/auth/login', {
        'email': email,
        'password': password,
      });

      if (response['requires_2fa'] == true) {
        return AuthResult(
          requires2FA: true,
          tempToken: response['temp_token'] as String?,
        );
      }

      final token = response['token'] as String?;
      final refreshToken = response['refresh_token'] as String?;
      if (token == null) {
        throw AuthException('Invalid response from server.');
      }

      await _saveTokens(token, refreshToken);
      final user = User.fromJson(response['user'] as Map<String, dynamic>);
      return AuthResult(user: user);
    } on ApiException catch (e) {
      throw AuthException(e.message);
    } catch (e) {
      throw AuthException('Login failed. Please check your credentials and try again.');
    }
  }

  Future<User> register(
    String email,
    String password,
    String name, {
    String? organizationName,
  }) async {
    try {
      final response = await _apiClient.post('/auth/register', {
        'email': email,
        'password': password,
        'name': name,
        if (organizationName != null && organizationName.isNotEmpty)
          'organization_name': organizationName,
      });

      final token = response['token'] as String?;
      final refreshToken = response['refresh_token'] as String?;
      if (token == null) {
        throw AuthException('Invalid response from server.');
      }

      await _saveTokens(token, refreshToken);
      return User.fromJson(response['user'] as Map<String, dynamic>);
    } on ApiException catch (e) {
      throw AuthException(e.message);
    } catch (e) {
      throw AuthException('Registration failed. Please try again.');
    }
  }

  Future<void> logout() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final token = prefs.getString(_tokenKey);
      if (token != null) {
        await _apiClient.post('/auth/logout', {});
      }
    } catch (_) {
      // Ignore logout API errors
    } finally {
      await _clearTokens();
    }
  }

  Future<User> verify2FA(String code) async {
    try {
      final response = await _apiClient.post('/auth/2fa/verify', {
        'code': code,
      });

      final token = response['token'] as String?;
      final refreshToken = response['refresh_token'] as String?;
      if (token == null) {
        throw AuthException('Invalid response from server.');
      }

      await _saveTokens(token, refreshToken);
      return User.fromJson(response['user'] as Map<String, dynamic>);
    } on ApiException catch (e) {
      throw AuthException(e.message);
    } catch (e) {
      throw AuthException('2FA verification failed. Please try again.');
    }
  }

  Future<String?> refreshToken() async {
    final prefs = await SharedPreferences.getInstance();
    final currentRefreshToken = prefs.getString(_refreshTokenKey);
    if (currentRefreshToken == null) return null;

    try {
      final response = await _apiClient.post('/auth/refresh', {
        'refresh_token': currentRefreshToken,
      });

      final newToken = response['token'] as String?;
      final newRefreshToken = response['refresh_token'] as String?;
      if (newToken != null) {
        await _saveTokens(newToken, newRefreshToken);
      }
      return newToken;
    } catch (e) {
      await _clearTokens();
      return null;
    }
  }

  Future<bool> isAuthenticated() async {
    final prefs = await SharedPreferences.getInstance();
    final token = prefs.getString(_tokenKey);
    if (token == null) return false;

    // Optionally validate token expiry here
    return true;
  }

  Future<User?> getCurrentUser() async {
    try {
      final response = await _apiClient.get('/auth/me');
      return User.fromJson(response);
    } catch (e) {
      return null;
    }
  }

  Future<void> _saveTokens(String token, String? refreshToken) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_tokenKey, token);
    if (refreshToken != null) {
      await prefs.setString(_refreshTokenKey, refreshToken);
    }
  }

  Future<void> _clearTokens() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_tokenKey);
    await prefs.remove(_refreshTokenKey);
  }
}
