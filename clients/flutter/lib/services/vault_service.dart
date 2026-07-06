import 'dart:convert';

import '../models/secret.dart';
import 'api_client.dart';

/// Service that performs CRUD operations for Vault secrets.
class VaultService {
  final ApiClient _apiClient;

  VaultService({ApiClient? apiClient})
      : _apiClient = apiClient ?? ApiClient(baseUrl: 'https://api.helix terminator.example.com');

  /// Fetches all secrets.
  Future<List<Secret>> getSecrets() async {
    final response = await _apiClient.get('/v1/vault/secrets');
    final data = response['data'] as List<dynamic>? ?? [];
    return data.map((json) => _secretFromJson(json as Map<String, dynamic>)).toList();
  }

  /// Creates a new secret.
  Future<Secret> createSecret({
    required String name,
    required String value,
    required String type,
    String? category,
    String? description,
  }) async {
    final response = await _apiClient.post('/v1/vault/secrets', {
      'name': name,
      'value': value,
      'type': type,
      'category': category,
      'description': description,
    });
    return _secretFromJson(response['data'] as Map<String, dynamic>);
  }

  /// Updates an existing secret.
  Future<Secret> updateSecret(
    String id, {
    String? name,
    String? value,
    String? type,
    String? category,
    String? description,
  }) async {
    final body = <String, dynamic>{};
    if (name != null) body['name'] = name;
    if (value != null) body['value'] = value;
    if (type != null) body['type'] = type;
    if (category != null) body['category'] = category;
    if (description != null) body['description'] = description;

    final response = await _apiClient.post('/v1/vault/secrets/$id', body);
    return _secretFromJson(response['data'] as Map<String, dynamic>);
  }

  /// Deletes a secret by [id].
  Future<void> deleteSecret(String id) async {
    await _apiClient.post('/v1/vault/secrets/$id/delete', {});
  }

  Secret _secretFromJson(Map<String, dynamic> json) {
    return Secret(
      id: json['id'] as String,
      name: json['name'] as String,
      type: json['type'] as String,
      category: json['category'] as String? ?? 'general',
      description: json['description'] as String?,
      createdAt: DateTime.parse(json['createdAt'] as String),
      updatedAt: json['updatedAt'] != null
          ? DateTime.parse(json['updatedAt'] as String)
          : null,
    );
  }
}
