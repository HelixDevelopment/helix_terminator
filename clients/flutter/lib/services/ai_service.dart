import 'api_client.dart';

class AiServiceException implements Exception {
  final String message;
  AiServiceException(this.message);
}

class AiService {
  final ApiClient _apiClient;

  AiService({required ApiClient apiClient}) : _apiClient = apiClient;

  Future<String> sendMessage(String message, {List<Map<String, String>>? history}) async {
    try {
      final response = await _apiClient.post('/api/v1/ai/chat', {
        'message': message,
        if (history != null) 'history': history,
      });
      return response['data'] as String? ?? '';
    } on ApiException catch (e) {
      throw AiServiceException(e.message);
    } catch (e) {
      throw AiServiceException('Failed to send message');
    }
  }

  Future<List<String>> getSuggestions(String context) async {
    try {
      final response = await _apiClient.post('/api/v1/ai/suggestions', {
        'context': context,
      });
      return (response['data'] as List<dynamic>? ?? []).map((e) => e as String).toList();
    } on ApiException catch (e) {
      throw AiServiceException(e.message);
    } catch (e) {
      throw AiServiceException('Failed to get suggestions');
    }
  }

  Future<String> explainCommand(String command) async {
    try {
      final response = await _apiClient.post('/api/v1/ai/explain', {
        'command': command,
      });
      return response['data'] as String? ?? '';
    } on ApiException catch (e) {
      throw AiServiceException(e.message);
    } catch (e) {
      throw AiServiceException('Failed to explain command');
    }
  }

  Future<String> generateCommand(String description) async {
    try {
      final response = await _apiClient.post('/api/v1/ai/generate', {
        'description': description,
      });
      return response['data'] as String? ?? '';
    } on ApiException catch (e) {
      throw AiServiceException(e.message);
    } catch (e) {
      throw AiServiceException('Failed to generate command');
    }
  }
}
