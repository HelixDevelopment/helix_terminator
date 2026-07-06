import '../models/port_forward.dart';
import 'api_client.dart';

class PortForwardService {
  final ApiClient _apiClient;

  PortForwardService({required ApiClient apiClient}) : _apiClient = apiClient;

  Future<List<PortForward>> getPortForwards() async {
    final response = await _apiClient.get('/api/v1/port-forwards');
    final data = response['data'] as List<dynamic>? ?? [];
    return data.map((json) => _portForwardFromJson(json as Map<String, dynamic>)).toList();
  }

  Future<PortForward> createPortForward({
    required String hostId,
    required int localPort,
    required int remotePort,
    String remoteHost = 'localhost',
  }) async {
    final response = await _apiClient.post('/api/v1/port-forwards', {
      'hostId': hostId,
      'localPort': localPort,
      'remotePort': remotePort,
      'remoteHost': remoteHost,
    });
    return _portForwardFromJson(response['data'] as Map<String, dynamic>);
  }

  Future<void> deletePortForward(String id) async {
    await _apiClient.post('/api/v1/port-forwards/$id/delete', {});
  }

  Future<void> startPortForward(String id) async {
    await _apiClient.post('/api/v1/port-forwards/$id/start', {});
  }

  Future<void> stopPortForward(String id) async {
    await _apiClient.post('/api/v1/port-forwards/$id/stop', {});
  }

  Future<List<Map<String, dynamic>>> getActiveConnections() async {
    final response = await _apiClient.get('/api/v1/port-forwards/active');
    return (response['data'] as List<dynamic>? ?? [])
        .map((e) => e as Map<String, dynamic>)
        .toList();
  }

  PortForward _portForwardFromJson(Map<String, dynamic> json) {
    return PortForward(
      id: json['id'] as String,
      hostId: json['hostId'] as String,
      localPort: json['localPort'] as int,
      remotePort: json['remotePort'] as int,
      remoteHost: json['remoteHost'] as String? ?? 'localhost',
      active: json['active'] as bool? ?? false,
    );
  }
}
