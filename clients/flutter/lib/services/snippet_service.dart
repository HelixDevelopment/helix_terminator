import '../models/snippet.dart';
import 'api_client.dart';

class SnippetService {
  final ApiClient _apiClient;

  SnippetService({required ApiClient apiClient}) : _apiClient = apiClient;

  Future<List<Snippet>> getSnippets({
    String? search,
    String? language,
    int limit = 50,
    int offset = 0,
  }) async {
    final queryParams = <String, String>{
      'limit': limit.toString(),
      'offset': offset.toString(),
    };
    if (search != null) queryParams['search'] = search;
    if (language != null) queryParams['language'] = language;

    final queryString = queryParams.entries
        .map((e) => '${Uri.encodeComponent(e.key)}=${Uri.encodeComponent(e.value)}')
        .join('&');

    final response = await _apiClient.get('/api/v1/snippets?$queryString');
    final data = response['data'] as List<dynamic>? ?? [];
    return data.map((json) => _snippetFromJson(json as Map<String, dynamic>)).toList();
  }

  Future<Snippet> getSnippet(String id) async {
    final response = await _apiClient.get('/api/v1/snippets/$id');
    return _snippetFromJson(response['data'] as Map<String, dynamic>);
  }

  Future<Snippet> createSnippet({
    required String title,
    required String content,
    required String language,
  }) async {
    final response = await _apiClient.post('/api/v1/snippets', {
      'title': title,
      'content': content,
      'language': language,
    });
    return _snippetFromJson(response['data'] as Map<String, dynamic>);
  }

  Future<Snippet> updateSnippet(
    String id, {
    String? title,
    String? content,
    String? language,
  }) async {
    final body = <String, dynamic>{};
    if (title != null) body['title'] = title;
    if (content != null) body['content'] = content;
    if (language != null) body['language'] = language;
    final response = await _apiClient.post('/api/v1/snippets/$id', body);
    return _snippetFromJson(response['data'] as Map<String, dynamic>);
  }

  Future<void> deleteSnippet(String id) async {
    await _apiClient.post('/api/v1/snippets/$id/delete', {});
  }

  Snippet _snippetFromJson(Map<String, dynamic> json) {
    return Snippet(
      id: json['id'] as String,
      title: json['title'] as String,
      content: json['content'] as String,
      language: json['language'] as String,
      createdAt: DateTime.parse(json['createdAt'] as String),
    );
  }
}
