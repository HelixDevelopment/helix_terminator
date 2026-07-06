import '../models/session.dart';
import 'api_client.dart';

class CollaborationService {
  final ApiClient _apiClient;

  CollaborationService({required ApiClient apiClient}) : _apiClient = apiClient;

  Future<List<Session>> getSessions() async {
    final response = await _apiClient.get('/api/v1/collaboration/sessions');
    final data = response['data'] as List<dynamic>? ?? [];
    return data.map((json) => _sessionFromJson(json as Map<String, dynamic>)).toList();
  }

  Future<Session> createSession({
    required String hostId,
    String? name,
  }) async {
    final response = await _apiClient.post('/api/v1/collaboration/sessions', {
      'hostId': hostId,
      if (name != null) 'name': name,
    });
    return _sessionFromJson(response['data'] as Map<String, dynamic>);
  }

  Future<Session> joinSession(String sessionId) async {
    final response = await _apiClient.post('/api/v1/collaboration/sessions/$sessionId/join', {});
    return _sessionFromJson(response['data'] as Map<String, dynamic>);
  }

  Future<void> leaveSession(String sessionId) async {
    await _apiClient.post('/api/v1/collaboration/sessions/$sessionId/leave', {});
  }

  Future<void> endSession(String sessionId) async {
    await _apiClient.post('/api/v1/collaboration/sessions/$sessionId/end', {});
  }

  Future<List<Map<String, dynamic>>> getParticipants(String sessionId) async {
    final response = await _apiClient.get('/api/v1/collaboration/sessions/$sessionId/participants');
    return (response['data'] as List<dynamic>? ?? [])
        .map((e) => e as Map<String, dynamic>)
        .toList();
  }

  Session _sessionFromJson(Map<String, dynamic> json) {
    return Session(
      id: json['id'] as String,
      hostId: json['hostId'] as String,
      startedAt: DateTime.parse(json['startedAt'] as String),
      endedAt: json['endedAt'] != null ? DateTime.parse(json['endedAt'] as String) : null,
      protocol: json['protocol'] as String? ?? 'ssh',
    );
  }
}
