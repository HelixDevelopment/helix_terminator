import '../models/audit_log.dart';
import 'api_client.dart';

class AuditService {
  final ApiClient _apiClient;

  AuditService({required ApiClient apiClient}) : _apiClient = apiClient;

  Future<List<AuditLog>> getAuditLogs({
    String? search,
    String? action,
    String? actor,
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
    if (action != null) queryParams['action'] = action;
    if (actor != null) queryParams['actor'] = actor;
    if (from != null) queryParams['from'] = from.toIso8601String();
    if (to != null) queryParams['to'] = to.toIso8601String();

    final queryString = queryParams.entries
        .map((e) => '${Uri.encodeComponent(e.key)}=${Uri.encodeComponent(e.value)}')
        .join('&');

    final response = await _apiClient.get('/api/v1/audit-logs?$queryString');
    final data = response['data'] as List<dynamic>? ?? [];
    return data.map((json) => _auditLogFromJson(json as Map<String, dynamic>)).toList();
  }

  Future<String> exportAuditLogs({
    String? action,
    DateTime? from,
    DateTime? to,
    String format = 'csv',
  }) async {
    final queryParams = <String, String>{
      'format': format,
    };
    if (action != null) queryParams['action'] = action;
    if (from != null) queryParams['from'] = from.toIso8601String();
    if (to != null) queryParams['to'] = to.toIso8601String();

    final queryString = queryParams.entries
        .map((e) => '${Uri.encodeComponent(e.key)}=${Uri.encodeComponent(e.value)}')
        .join('&');

    final response = await _apiClient.get('/api/v1/audit-logs/export?$queryString');
    return response['data'] as String? ?? '';
  }

  AuditLog _auditLogFromJson(Map<String, dynamic> json) {
    return AuditLog(
      id: json['id'] as String,
      actor: json['actor'] as String,
      action: json['action'] as String,
      target: json['target'] as String,
      timestamp: DateTime.parse(json['timestamp'] as String),
    );
  }
}
