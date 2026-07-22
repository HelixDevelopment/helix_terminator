// Real tests for NotificationBloc — replaces the former `expect(true,
// isTrue)` stub (§11.4/§11.4.27). Drives every real event through the real
// bloc against a mocked NotificationService, including the state-guarded
// client-side search filter and the mark-as-read -> auto-reload chain.
//
// `models.Notification` is imported with a prefix because Flutter's own
// framework also exports a `Notification` type (package:flutter/widgets.dart,
// transitively pulled in by flutter_test) — the same disambiguation the
// production notification_service.dart already uses.

import 'package:bloc_test/bloc_test.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';
import 'package:helix_terminator/bloc/notification_bloc.dart';
import 'package:helix_terminator/models/notification.dart' as models;
import 'package:helix_terminator/services/notification_service.dart';

class MockNotificationService extends Mock implements NotificationService {}

void main() {
  late MockNotificationService service;

  models.Notification notification({required String id, bool read = false}) => models.Notification(
        id: id,
        title: 'Title $id',
        body: 'Body $id',
        read: read,
        createdAt: DateTime.utc(2026, 1, 1),
      );

  setUp(() {
    service = MockNotificationService();
  });

  group('NotificationBloc', () {
    blocTest<NotificationBloc, NotificationState>(
      'NotificationListRequested success emits [Loading, Loaded] with the real list',
      build: () {
        when(() => service.getNotifications())
            .thenAnswer((_) async => [notification(id: '1'), notification(id: '2')]);
        return NotificationBloc(service: service);
      },
      act: (bloc) => bloc.add(NotificationListRequested()),
      expect: () => [
        isA<NotificationLoading>(),
        isA<NotificationLoaded>().having((s) => s.notifications.length, 'notifications.length', 2),
      ],
    );

    blocTest<NotificationBloc, NotificationState>(
      'NotificationListRequested failure emits [Loading, Error]',
      build: () {
        when(() => service.getNotifications()).thenThrow(NotificationServiceException('boom'));
        return NotificationBloc(service: service);
      },
      act: (bloc) => bloc.add(NotificationListRequested()),
      expect: () => [isA<NotificationLoading>(), isA<NotificationError>()],
    );

    blocTest<NotificationBloc, NotificationState>(
      'NotificationFilterChanged forwards type + unreadOnly to the service and records them on state',
      build: () {
        when(
          () => service.getNotifications(
            type: any(named: 'type'),
            unreadOnly: any(named: 'unreadOnly'),
          ),
        ).thenAnswer((_) async => [notification(id: '1', read: false)]);
        return NotificationBloc(service: service);
      },
      act: (bloc) => bloc.add(NotificationFilterChanged(type: 'security', unreadOnly: true)),
      expect: () => [
        isA<NotificationLoading>(),
        isA<NotificationLoaded>()
            .having((s) => s.filterType, 'filterType', 'security')
            .having((s) => s.unreadOnly, 'unreadOnly', isTrue),
      ],
      verify: (_) {
        verify(() => service.getNotifications(type: 'security', unreadOnly: true)).called(1);
      },
    );

    blocTest<NotificationBloc, NotificationState>(
      'NotificationMarkAsRead success emits ActionSuccess then re-triggers the real list reload',
      build: () {
        when(() => service.markAsRead('n-1')).thenAnswer((_) async {});
        when(() => service.getNotifications()).thenAnswer((_) async => [notification(id: 'n-1', read: true)]);
        return NotificationBloc(service: service);
      },
      act: (bloc) => bloc.add(NotificationMarkAsRead('n-1')),
      expect: () => [
        isA<NotificationActionSuccess>().having((s) => s.message, 'message', 'Marked as read'),
        isA<NotificationLoading>(),
        isA<NotificationLoaded>().having((s) => s.notifications.single.read, 'notifications[0].read', isTrue),
      ],
      verify: (_) {
        verify(() => service.markAsRead('n-1')).called(1);
      },
    );

    blocTest<NotificationBloc, NotificationState>(
      'NotificationMarkAllAsRead success emits ActionSuccess then reloads',
      build: () {
        when(() => service.markAllAsRead()).thenAnswer((_) async {});
        when(() => service.getNotifications()).thenAnswer((_) async => const []);
        return NotificationBloc(service: service);
      },
      act: (bloc) => bloc.add(NotificationMarkAllAsRead()),
      expect: () => [
        isA<NotificationActionSuccess>().having((s) => s.message, 'message', 'All marked as read'),
        isA<NotificationLoading>(),
        isA<NotificationLoaded>(),
      ],
    );

    blocTest<NotificationBloc, NotificationState>(
      'NotificationDelete failure emits Error only (no reload dispatched)',
      build: () {
        when(() => service.deleteNotification('n-9')).thenThrow(NotificationServiceException('gone'));
        return NotificationBloc(service: service);
      },
      act: (bloc) => bloc.add(NotificationDelete('n-9')),
      expect: () => [isA<NotificationError>()],
      verify: (_) {
        verifyNever(() => service.getNotifications());
      },
    );

    blocTest<NotificationBloc, NotificationState>(
      'NotificationSearchChanged is a NO-OP when not currently Loaded '
      '(proves the state-guard is real, not a stub)',
      build: () => NotificationBloc(service: service),
      act: (bloc) => bloc.add(NotificationSearchChanged('urgent')),
      expect: () => <Matcher>[],
    );

    blocTest<NotificationBloc, NotificationState>(
      'NotificationSearchChanged while Loaded updates only the searchQuery, keeping the same list',
      seed: () => NotificationLoaded([notification(id: '1')], filterType: 'security'),
      build: () => NotificationBloc(service: service),
      act: (bloc) => bloc.add(NotificationSearchChanged('urgent')),
      expect: () => [
        isA<NotificationLoaded>()
            .having((s) => s.searchQuery, 'searchQuery', 'urgent')
            .having((s) => s.filterType, 'filterType (preserved)', 'security')
            .having((s) => s.notifications.length, 'notifications.length (preserved)', 1),
      ],
    );
  });
}
