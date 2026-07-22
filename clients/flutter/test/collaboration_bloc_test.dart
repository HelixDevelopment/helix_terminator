// Real tests for CollaborationBloc — replaces the former `expect(true,
// isTrue)` stub (§11.4/§11.4.27). Drives every real event through the real
// bloc against a mocked CollaborationService, asserting the real emitted
// CollaborationState sequence, including the state-guarded negative path
// (participants update is a no-op when no session is active).

import 'package:bloc_test/bloc_test.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';
import 'package:helix_terminator/bloc/collaboration_bloc.dart';
import 'package:helix_terminator/models/session.dart';
import 'package:helix_terminator/services/collaboration_service.dart';

class MockCollaborationService extends Mock implements CollaborationService {}

void main() {
  late MockCollaborationService service;

  Session session({String id = 's-1', String hostId = 'host-1'}) => Session(
        id: id,
        hostId: hostId,
        startedAt: DateTime.utc(2026, 1, 1),
        protocol: 'ssh',
      );

  setUp(() {
    service = MockCollaborationService();
  });

  group('CollaborationBloc', () {
    blocTest<CollaborationBloc, CollaborationState>(
      'CollaborationListRequested success emits [Loading, ListLoaded] with the real sessions',
      build: () {
        when(() => service.getSessions()).thenAnswer((_) async => [session(id: 's-1'), session(id: 's-2')]);
        return CollaborationBloc(service: service);
      },
      act: (bloc) => bloc.add(CollaborationListRequested()),
      expect: () => [
        isA<CollaborationLoading>(),
        isA<CollaborationListLoaded>().having((s) => s.sessions.length, 'sessions.length', 2),
      ],
    );

    blocTest<CollaborationBloc, CollaborationState>(
      'CollaborationListRequested failure emits [Loading, Error]',
      build: () {
        when(() => service.getSessions()).thenThrow(Exception('network down'));
        return CollaborationBloc(service: service);
      },
      act: (bloc) => bloc.add(CollaborationListRequested()),
      expect: () => [isA<CollaborationLoading>(), isA<CollaborationError>()],
    );

    blocTest<CollaborationBloc, CollaborationState>(
      'CollaborationCreateSession success emits [ActionSuccess, Active] with the created session',
      build: () {
        when(() => service.createSession(hostId: 'host-1', name: 'Pair debug'))
            .thenAnswer((_) async => session(id: 'new-session'));
        return CollaborationBloc(service: service);
      },
      act: (bloc) => bloc.add(CollaborationCreateSession(hostId: 'host-1', name: 'Pair debug')),
      expect: () => [
        isA<CollaborationActionSuccess>().having((s) => s.message, 'message', 'Session created'),
        isA<CollaborationActive>().having(
          (s) => (s.session as Session).id,
          'session.id',
          'new-session',
        ),
      ],
    );

    blocTest<CollaborationBloc, CollaborationState>(
      'CollaborationCreateSession failure emits [Error] only',
      build: () {
        when(() => service.createSession(hostId: any(named: 'hostId'), name: any(named: 'name')))
            .thenThrow(Exception('host offline'));
        return CollaborationBloc(service: service);
      },
      act: (bloc) => bloc.add(CollaborationCreateSession(hostId: 'host-9')),
      expect: () => [isA<CollaborationError>()],
    );

    blocTest<CollaborationBloc, CollaborationState>(
      'CollaborationJoinSession success emits [ActionSuccess, Active] carrying real participants',
      build: () {
        when(() => service.joinSession('s-1')).thenAnswer((_) async => session(id: 's-1'));
        when(() => service.getParticipants('s-1')).thenAnswer(
          (_) async => [
            {'userId': 'u-1', 'name': 'Alice'},
          ],
        );
        return CollaborationBloc(service: service);
      },
      act: (bloc) => bloc.add(CollaborationJoinSession('s-1')),
      expect: () => [
        isA<CollaborationActionSuccess>().having((s) => s.message, 'message', 'Joined session'),
        isA<CollaborationActive>()
            .having((s) => (s.session as Session).id, 'session.id', 's-1')
            .having((s) => s.participants.single['name'], 'participants[0].name', 'Alice'),
      ],
    );

    blocTest<CollaborationBloc, CollaborationState>(
      'CollaborationParticipantsRequested is a NO-OP when no session is active '
      '(real state-guard behaviour, proves the bloc does not blindly emit)',
      build: () {
        when(() => service.getParticipants('s-1')).thenAnswer((_) async => const []);
        return CollaborationBloc(service: service);
      },
      act: (bloc) => bloc.add(CollaborationParticipantsRequested('s-1')),
      expect: () => <Matcher>[],
      verify: (_) {
        // The service IS called (the bloc always fetches); it is the emit
        // that is guarded on `state is CollaborationActive`.
        verify(() => service.getParticipants('s-1')).called(1);
      },
    );

    blocTest<CollaborationBloc, CollaborationState>(
      'CollaborationParticipantsRequested refreshes participants on the active session '
      'while preserving the same session object',
      seed: () => CollaborationActive(session(id: 's-1'), participants: const []),
      build: () {
        when(() => service.getParticipants('s-1')).thenAnswer(
          (_) async => [
            {'userId': 'u-2', 'name': 'Bob'},
          ],
        );
        return CollaborationBloc(service: service);
      },
      act: (bloc) => bloc.add(CollaborationParticipantsRequested('s-1')),
      expect: () => [
        isA<CollaborationActive>().having(
          (s) => s.participants.single['name'],
          'participants[0].name',
          'Bob',
        ),
      ],
    );

    blocTest<CollaborationBloc, CollaborationState>(
      'CollaborationLeaveSession while active leaves + re-requests the list '
      '(real chained-event behaviour: ActionSuccess then Loading/ListLoaded)',
      seed: () => CollaborationActive(session(id: 's-1')),
      build: () {
        when(() => service.leaveSession('s-1')).thenAnswer((_) async {});
        when(() => service.getSessions()).thenAnswer((_) async => [session(id: 's-2')]);
        return CollaborationBloc(service: service);
      },
      act: (bloc) => bloc.add(CollaborationLeaveSession()),
      expect: () => [
        isA<CollaborationActionSuccess>().having((s) => s.message, 'message', 'Left session'),
        isA<CollaborationLoading>(),
        isA<CollaborationListLoaded>().having((s) => s.sessions.length, 'sessions.length', 1),
      ],
      verify: (_) {
        verify(() => service.leaveSession('s-1')).called(1);
      },
    );

    blocTest<CollaborationBloc, CollaborationState>(
      'CollaborationEndSession ends + re-requests the list',
      build: () {
        when(() => service.endSession('s-1')).thenAnswer((_) async {});
        when(() => service.getSessions()).thenAnswer((_) async => const []);
        return CollaborationBloc(service: service);
      },
      act: (bloc) => bloc.add(CollaborationEndSession('s-1')),
      expect: () => [
        isA<CollaborationActionSuccess>().having((s) => s.message, 'message', 'Session ended'),
        isA<CollaborationLoading>(),
        isA<CollaborationListLoaded>().having((s) => s.sessions, 'sessions', isEmpty),
      ],
    );
  });
}
