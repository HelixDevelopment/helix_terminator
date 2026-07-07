import '../models/notification.dart' as models;
import 'api_client.dart';

class NotificationServiceException implements Exception {
  final String message;
  NotificationServiceException(this.message);
}

class NotificationService {
  final ApiClient _apiClient;

  NotificationService({required ApiClient apiClient}) : _apiClient = apiClient;

  Future<List<models.Notification>> getNotifications({
    String? type,
    bool? unreadOnly,
    int limit = 50,
    int offset = 0,
  }) async {
    try {
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
    } on ApiException catch (e) {
      throw NotificationServiceException(e.message);
    } catch (e) {
      throw NotificationServiceException('Failed to load notifications');
    }
  }

  Future<models.Notification> getNotification(String id) async {
    try {
      final response = await _apiClient.get('/api/v1/notifications/$id');
      return _notificationFromJson(response['data'] as Map<String, dynamic>);
    } on ApiException catch (e) {
      throw NotificationServiceException(e.message);
    } catch (e) {
      throw NotificationServiceException('Failed to load notification');
    }
  }

  Future<void> markAsRead(String id) async {
    try {
      await _apiClient.post('/api/v1/notifications/$id/read', {});
    } on ApiException catch (e) {
      throw NotificationServiceException(e.message);
    } catch (e) {
      throw NotificationServiceException('Failed to mark as read');
    }
  }

  Future<void> markAllAsRead() async {
    try {
      await _apiClient.post('/api/v1/notifications/read-all', {});
    } on ApiException catch (e) {
      throw NotificationServiceException(e.message);
    } catch (e) {
      throw NotificationServiceException('Failed to mark all as read');
    }
  }

  Future<void> deleteNotification(String id) async {
    try {
      await _apiClient.post('/api/v1/notifications/$id/delete', {});
    } on ApiException catch (e) {
      throw NotificationServiceException(e.message);
    } catch (e) {
      throw NotificationServiceException('Failed to delete notification');
    }
  }

  Future<int> getUnreadCount() async {
    try {
      final response = await _apiClient.get('/api/v1/notifications/unread-count');
      return response['data'] as int? ?? 0;
    } on ApiException catch (e) {
      throw NotificationServiceException(e.message);
    } catch (e) {
      throw NotificationServiceException('Failed to get unread count');
    }
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
