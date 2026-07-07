import 'dart:convert';
import 'package:http/http.dart' as http;

import '../models/workspace.dart';
import 'api_client.dart';

class WorkspaceServiceException implements Exception {
  final String message;
  WorkspaceServiceException(this.message);
}

/// Service that performs CRUD operations for Workspaces.
class WorkspaceService {
  final ApiClient _apiClient;

  WorkspaceService({ApiClient? apiClient})
      : _apiClient = apiClient ?? ApiClient(baseUrl: 'https://api.helix-terminator.example.com');

  /// Fetches all workspaces.
  Future<List<Workspace>> getWorkspaces() async {
    try {
      final response = await _apiClient.get('/v1/workspaces');
      final data = response['data'] as List<dynamic>? ?? [];
      return data.map((json) => _workspaceFromJson(json as Map<String, dynamic>)).toList();
    } on ApiException catch (e) {
      throw WorkspaceServiceException(e.message);
    } catch (e) {
      throw WorkspaceServiceException('Failed to load workspaces');
    }
  }

  /// Creates a new workspace.
  Future<Workspace> createWorkspace({
    required String name,
    String? description,
    List<String> hostIds = const [],
  }) async {
    try {
      final response = await _apiClient.post('/v1/workspaces', {
        'name': name,
        'description': description,
        'hostIds': hostIds,
      });
      return _workspaceFromJson(response['data'] as Map<String, dynamic>);
    } on ApiException catch (e) {
      throw WorkspaceServiceException(e.message);
    } catch (e) {
      throw WorkspaceServiceException('Failed to create workspace');
    }
  }

  /// Updates an existing workspace.
  Future<Workspace> updateWorkspace(
    String id, {
    String? name,
    String? description,
    List<String>? hostIds,
  }) async {
    try {
      final body = <String, dynamic>{};
      if (name != null) body['name'] = name;
      if (description != null) body['description'] = description;
      if (hostIds != null) body['hostIds'] = hostIds;

      final response = await _apiClient.post('/v1/workspaces/$id', body);
      return _workspaceFromJson(response['data'] as Map<String, dynamic>);
    } on ApiException catch (e) {
      throw WorkspaceServiceException(e.message);
    } catch (e) {
      throw WorkspaceServiceException('Failed to update workspace');
    }
  }

  /// Deletes a workspace by [id].
  Future<void> deleteWorkspace(String id) async {
    try {
      await _apiClient.post('/v1/workspaces/$id/delete', {});
    } on ApiException catch (e) {
      throw WorkspaceServiceException(e.message);
    } catch (e) {
      throw WorkspaceServiceException('Failed to delete workspace');
    }
  }

  /// Adds a member to a workspace.
  Future<void> addMember(String workspaceId, String userId, {String role = 'member'}) async {
    try {
      await _apiClient.post('/v1/workspaces/$workspaceId/members', {
        'userId': userId,
        'role': role,
      });
    } on ApiException catch (e) {
      throw WorkspaceServiceException(e.message);
    } catch (e) {
      throw WorkspaceServiceException('Failed to add member');
    }
  }

  /// Removes a member from a workspace.
  Future<void> removeMember(String workspaceId, String userId) async {
    try {
      await _apiClient.post('/v1/workspaces/$workspaceId/members/$userId/remove', {});
    } on ApiException catch (e) {
      throw WorkspaceServiceException(e.message);
    } catch (e) {
      throw WorkspaceServiceException('Failed to remove member');
    }
  }

  Workspace _workspaceFromJson(Map<String, dynamic> json) {
    return Workspace(
      id: json['id'] as String,
      name: json['name'] as String,
      description: json['description'] as String?,
      hostIds: (json['hostIds'] as List<dynamic>?)?.cast<String>() ?? [],
      memberIds: (json['memberIds'] as List<dynamic>?)?.cast<String>() ?? [],
      createdAt: DateTime.parse(json['createdAt'] as String),
      updatedAt: json['updatedAt'] != null
          ? DateTime.parse(json['updatedAt'] as String)
          : null,
    );
  }
}
