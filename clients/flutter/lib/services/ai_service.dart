import 'api_client.dart';

class AiService {
  final ApiClient _apiClient;

  AiService({required ApiClient apiClient}) : _apiClient = apiClient;

  Future<String> sendMessage(String message, {List<Map<String, String>>? history}) async {
    final response = await _apiClient.post('/api/v1/ai/chat', {
      'message': message,
      if (history != null) 'history': history,
    });
    return response['data'] as String? ?? '';
  }

  Future<List<String>> getSuggestions(String context) async {
    final response = await _apiClient.post('/api/v1/ai/suggestions', {
      'context': context,
    });
    return (response['data'] as List<dynamic>? ?? []).map((e) => e as String).toList();
  }

  Future<String> explainCommand(String command) async {
    final response = await _apiClient.post('/api/v1/ai/explain', {
      'command': command,
    });
    return response['data'] as String? ?? '';
  }

  Future<String> generateCommand(String description) async {
    final response = await _apiClient.post('/api/v1/ai/generate', {
      'description': description,
    });
    return response['data'] as String? ?? '';
  }
}
