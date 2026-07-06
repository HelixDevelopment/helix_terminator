import '../models/recording.dart';
import 'api_client.dart';

class RecordingService {
  final ApiClient _apiClient;

  RecordingService({required ApiClient apiClient}) : _apiClient = apiClient;

  Future<List<Recording>> getRecordings({
    String? search,
    String? sessionId,
    DateTime? from,
    DateTime? to,
    int limit = 50,
    int offset = 0,
  }) async {
    final queryParams = <String, String>{
      'limit': limit.toString(),
      'offset': offset.toString(),
    };
    if (search != null) queryParams['search'] = search;
    if (sessionId != null) queryParams['sessionId'] = sessionId;
    if (from != null) queryParams['from'] = from.toIso8601String();
    if (to != null) queryParams['to'] = to.toIso8601String();

    final queryString = queryParams.entries
        .map((e) => '${Uri.encodeComponent(e.key)}=${Uri.encodeComponent(e.value)}')
        .join('&');

    final response = await _apiClient.get('/api/v1/recordings?$queryString');
    final data = response['data'] as List<dynamic>? ?? [];
    return data.map((json) => _recordingFromJson(json as Map<String, dynamic>)).toList();
  }

  Future<Recording> getRecording(String id) async {
    final response = await _apiClient.get('/api/v1/recordings/$id');
    return _recordingFromJson(response['data'] as Map<String, dynamic>);
  }

  Future<void> deleteRecording(String id) async {
    await _apiClient.post('/api/v1/recordings/$id/delete', {});
  }

  Future<String> getRecordingUrl(String id) async {
    final response = await _apiClient.get('/api/v1/recordings/$id/url');
    return response['data'] as String? ?? '';
  }

  Recording _recordingFromJson(Map<String, dynamic> json) {
    return Recording(
      id: json['id'] as String,
      sessionId: json['sessionId'] as String,
      title: json['title'] as String,
      duration: Duration(seconds: json['durationSeconds'] as int? ?? 0),
      createdAt: DateTime.parse(json['createdAt'] as String),
    );
  }
}
