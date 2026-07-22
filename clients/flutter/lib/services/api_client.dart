import 'package:http/http.dart' as http;
import 'dart:convert';
import 'auth_service.dart';

// TODO: replace with real API base URL and add auth interceptors

class ApiException implements Exception {
  final String message;
  final int? statusCode;

  ApiException(this.message, {this.statusCode});

  // Without this override, any `'$e'` / `e.toString()` call on a caught
  // ApiException (the pattern every *Bloc error handler uses) renders the
  // useless default `Instance of 'ApiException'` instead of the real
  // message — discovered via test/api_client_test.dart & the *_bloc_test.dart
  // suite's failure-path assertions (§11.4.1/§11.4.4 test-interrupt-on-
  // discovery: root-caused and fixed here rather than the tests weakened to
  // match the broken output).
  @override
  String toString() => message;
}

class ApiClient {
  final String baseUrl;
  final http.Client _client;
  final AuthService? _authService;

  /// [httpClient] is an injectable seam for tests (e.g. `http.testing.MockClient`)
  /// so requests can be exercised without a real network call. Production
  /// callers omit it and get a real `http.Client()`, unchanged from before.
  ApiClient({required this.baseUrl, AuthService? authService, http.Client? httpClient})
      : _client = httpClient ?? http.Client(),
        _authService = authService;

  Future<Map<String, dynamic>> get(String path) async {
    final response = await _client.get(
      Uri.parse('$baseUrl$path'),
      headers: await _headers(),
    );
    return _handleResponse(response);
  }

  Future<Map<String, dynamic>> post(String path, Map<String, dynamic> body) async {
    final response = await _client.post(
      Uri.parse('$baseUrl$path'),
      headers: await _headers(),
      body: jsonEncode(body),
    );
    return _handleResponse(response);
  }

  Future<Map<String, dynamic>> put(String path, Map<String, dynamic> body) async {
    final response = await _client.put(
      Uri.parse('$baseUrl$path'),
      headers: await _headers(),
      body: jsonEncode(body),
    );
    return _handleResponse(response);
  }

  Future<Map<String, dynamic>> delete(String path) async {
    final response = await _client.delete(
      Uri.parse('$baseUrl$path'),
      headers: await _headers(),
    );
    return _handleResponse(response);
  }

  Future<Map<String, String>> _headers() async {
    final headers = {'Content-Type': 'application/json'};
    final token = await _authService?.getToken();
    if (token != null && token.isNotEmpty) {
      headers['Authorization'] = 'Bearer $token';
    }
    return headers;
  }

  Map<String, dynamic> _handleResponse(http.Response response) {
    if (response.statusCode >= 200 && response.statusCode < 300) {
      if (response.body.isEmpty) return {};
      return jsonDecode(response.body) as Map<String, dynamic>;
    }
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    throw ApiException(
      body['message'] ?? body['error'] ?? 'Request failed',
      statusCode: response.statusCode,
    );
  }

  void dispose() {
    _client.close();
  }
}
