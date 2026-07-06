import '../models/notification.dart' as models;
import 'api_client.dart';

class NotificationService {
  final ApiClient _apiClient;

  NotificationService({required ApiClient apiClient}) : _apiClient = apiClient;

  Future<List<models.Notification>> getNotifications({
    String? type,
    bool? unreadOnly,
    int limit = 50,
    int offset = 0,
  }) async {
    final queryParams = <String, String>{
      'limit': limit.toString(),
      'offset': offset.toString(),
    };
    if (type != null) queryParams['type'] = type;
    if (unreadOnly != null) queryParams['unread'] = unreadOnly.toString();

    final queryString = queryParams.entries
        .map((e) => '${Uri.encodeComponent(e.key)}=${Uri.encodeComponent(e.value)}')
        .join('&');

    final response = await _apiClient.get('/api/v1/notifications?$queryString');
    final data = response['data'] as List<dynamic>? ?? [];
    return data.map((json) => _notificationFromJson(json as Map<String, dynamic>)).toList();
  }

  Future<models.Notification> getNotification(String id) async {
    final response = await _apiClient.get('/api/v1/notifications/$id');
    return _notificationFromJson(response['data'] as Map<String, dynamic>);
  }

  Future<void> markAsRead(String id) async {
    await _apiClient.post('/api/v1/notifications/$id/read', {});
  }

  Future<void> markAllAsRead() async {
    await _apiClient.post('/api/v1/notifications/read-all', {});
  }

  Future<void> deleteNotification(String id) async {
    await _apiClient.post('/api/v1/notifications/$id/delete', {});
  }

  Future<int> getUnreadCount() async {
    final response = await _apiClient.get('/api/v1/notifications/unread-count');
    return response['data'] as int? ?? 0;
  }

  models.Notification _notificationFromJson(Map<String, dynamic> json) {
    return models.Notification(
      id: json['id'] as String,
      title: json['title'] as String,
      body: json['body'] as String,
      read: json['read'] as bool? ?? false,
      createdAt: DateTime.parse(json['createdAt'] as String),
    );
  }
}
