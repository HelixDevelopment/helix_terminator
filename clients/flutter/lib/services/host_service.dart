import '../models/host.dart';
import 'api_client.dart';

class HostService {
  final ApiClient _apiClient;

  HostService({required ApiClient apiClient}) : _apiClient = apiClient;

  Future<List<Host>> getHosts() async {
    final response = await _apiClient.get('/hosts');
    final data = response['data'] as List<dynamic>? ?? [];
    return data.map((json) => _hostFromJson(json as Map<String, dynamic>)).toList();
  }

  Future<Host> getHostById(String id) async {
    final response = await _apiClient.get('/hosts/$id');
    final data = response['data'] as Map<String, dynamic>;
    return _hostFromJson(data);
  }

  Future<Host> createHost(Host host) async {
    final response = await _apiClient.post('/hosts', _hostToJson(host));
    final data = response['data'] as Map<String, dynamic>;
    return _hostFromJson(data);
  }

  Future<Host> updateHost(String id, Host host) async {
    final response = await _apiClient.put('/hosts/$id', _hostToJson(host));
    final data = response['data'] as Map<String, dynamic>;
    return _hostFromJson(data);
  }

  Future<void> deleteHost(String id) async {
    await _apiClient.delete('/hosts/$id');
  }

  Host _hostFromJson(Map<String, dynamic> json) {
    return Host(
      id: json['id'] as String,
      name: json['name'] as String,
      address: json['address'] as String,
      port: json['port'] as int? ?? 22,
      username: json['username'] as String?,
      tags: (json['tags'] as List<dynamic>?)?.map((e) => e as String).toList() ?? [],
      createdAt: DateTime.parse(json['createdAt'] as String),
      status: json['status'] as String? ?? 'unknown',
      organizationId: json['organizationId'] as String?,
      authMethod: json['authMethod'] as String? ?? 'password',
    );
  }

  Map<String, dynamic> _hostToJson(Host host) {
    return {
      'id': host.id,
      'name': host.name,
      'address': host.address,
      'port': host.port,
      'username': host.username,
      'tags': host.tags,
      'createdAt': host.createdAt.toIso8601String(),
      'status': host.status,
      'organizationId': host.organizationId,
      'authMethod': host.authMethod,
    };
  }
}
