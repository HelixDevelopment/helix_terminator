// Real tests for WorkspaceBloc — replaces the former `expect(true, isTrue)`
// stub (§11.4/§11.4.27). Drives every real event through the real bloc
// against a mocked WorkspaceService, including the one behaviour a stub can
// never catch: WorkspaceCreateRequested's extra trailing Loaded emission is
// CONDITIONAL on whether the bloc was already Loaded before the event (see
// `previousState is WorkspaceLoaded` in workspace_bloc.dart) — this suite
// proves BOTH branches of that condition for real.

import 'package:bloc_test/bloc_test.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';
import 'package:helix_terminator/bloc/workspace_bloc.dart';
import 'package:helix_terminator/models/workspace.dart';
import 'package:helix_terminator/services/workspace_service.dart';

class MockWorkspaceService extends Mock implements WorkspaceService {}

void main() {
  late MockWorkspaceService workspaceService;

  Workspace workspace({required String id, String name = 'infra'}) => Workspace(
        id: id,
        name: name,
        createdAt: DateTime.utc(2026, 1, 1),
      );

  setUpAll(() {
    // mocktail requires a registered fallback value for any() used with a
    // non-builtin argument type (List<String> hostIds, below).
    registerFallbackValue(<String>[]);
  });

  setUp(() {
    workspaceService = MockWorkspaceService();
  });

  group('WorkspaceBloc', () {
    blocTest<WorkspaceBloc, WorkspaceState>(
      'WorkspaceLoadRequested success emits [Loading, Loaded] with the real workspaces',
      build: () {
        when(() => workspaceService.getWorkspaces())
            .thenAnswer((_) async => [workspace(id: '1'), workspace(id: '2')]);
        return WorkspaceBloc(workspaceService: workspaceService);
      },
      act: (bloc) => bloc.add(WorkspaceLoadRequested()),
      expect: () => [
        isA<WorkspaceLoading>(),
        isA<WorkspaceLoaded>().having((s) => s.workspaces.map((w) => w.id), 'workspace ids', ['1', '2']),
      ],
    );

    blocTest<WorkspaceBloc, WorkspaceState>(
      'WorkspaceLoadRequested failure emits [Loading, Error]',
      build: () {
        when(() => workspaceService.getWorkspaces())
            .thenThrow(WorkspaceServiceException('forbidden'));
        return WorkspaceBloc(workspaceService: workspaceService);
      },
      act: (bloc) => bloc.add(WorkspaceLoadRequested()),
      expect: () => [
        isA<WorkspaceLoading>(),
        isA<WorkspaceError>().having((s) => s.message, 'message', contains('forbidden')),
      ],
    );

    blocTest<WorkspaceBloc, WorkspaceState>(
      'WorkspaceCreateRequested from a FRESH (non-Loaded) bloc emits the SHORT sequence: '
      '[Loading, Loaded, OperationSuccess] — no trailing duplicate Loaded',
      build: () {
        when(() => workspaceService.createWorkspace(name: 'new-ws', description: null, hostIds: const []))
            .thenAnswer((_) async => workspace(id: 'new-1', name: 'new-ws'));
        when(() => workspaceService.getWorkspaces()).thenAnswer((_) async => [workspace(id: 'new-1', name: 'new-ws')]);
        return WorkspaceBloc(workspaceService: workspaceService);
      },
      act: (bloc) => bloc.add(WorkspaceCreateRequested(name: 'new-ws')),
      expect: () => [
        isA<WorkspaceLoading>(),
        isA<WorkspaceLoaded>().having((s) => s.workspaces.single.name, 'workspaces[0].name', 'new-ws'),
        isA<WorkspaceOperationSuccess>().having((s) => s.message, 'message', 'Workspace created'),
      ],
    );

    blocTest<WorkspaceBloc, WorkspaceState>(
      'WorkspaceCreateRequested when the bloc was ALREADY Loaded emits the LONG sequence: '
      '[Loading, Loaded, OperationSuccess, Loaded] — the real conditional trailing re-emit',
      seed: () => WorkspaceLoaded([workspace(id: 'old-1', name: 'existing')]),
      build: () {
        when(() => workspaceService.createWorkspace(name: 'new-ws', description: null, hostIds: const []))
            .thenAnswer((_) async => workspace(id: 'new-1', name: 'new-ws'));
        when(() => workspaceService.getWorkspaces())
            .thenAnswer((_) async => [workspace(id: 'old-1', name: 'existing'), workspace(id: 'new-1', name: 'new-ws')]);
        return WorkspaceBloc(workspaceService: workspaceService);
      },
      act: (bloc) => bloc.add(WorkspaceCreateRequested(name: 'new-ws')),
      expect: () => [
        isA<WorkspaceLoading>(),
        isA<WorkspaceLoaded>().having((s) => s.workspaces.length, 'workspaces.length', 2),
        isA<WorkspaceOperationSuccess>(),
        isA<WorkspaceLoaded>().having((s) => s.workspaces.length, 'workspaces.length (re-emitted)', 2),
      ],
    );

    blocTest<WorkspaceBloc, WorkspaceState>(
      'WorkspaceCreateRequested failure emits [Loading, Error] and never calls getWorkspaces',
      build: () {
        when(() => workspaceService.createWorkspace(
              name: any(named: 'name'),
              description: any(named: 'description'),
              hostIds: any(named: 'hostIds'),
            )).thenThrow(WorkspaceServiceException('quota exceeded'));
        return WorkspaceBloc(workspaceService: workspaceService);
      },
      act: (bloc) => bloc.add(WorkspaceCreateRequested(name: 'too-many')),
      expect: () => [
        isA<WorkspaceLoading>(),
        isA<WorkspaceError>().having((s) => s.message, 'message', contains('quota exceeded')),
      ],
      verify: (_) {
        verifyNever(() => workspaceService.getWorkspaces());
      },
    );

    blocTest<WorkspaceBloc, WorkspaceState>(
      'WorkspaceUpdateRequested success updates then reloads: '
      '[Loading, Loaded, OperationSuccess, Loaded]',
      build: () {
        when(() => workspaceService.updateWorkspace('ws-1', name: 'renamed', description: null, hostIds: null))
            .thenAnswer((_) async => workspace(id: 'ws-1', name: 'renamed'));
        when(() => workspaceService.getWorkspaces()).thenAnswer((_) async => [workspace(id: 'ws-1', name: 'renamed')]);
        return WorkspaceBloc(workspaceService: workspaceService);
      },
      act: (bloc) => bloc.add(WorkspaceUpdateRequested(id: 'ws-1', name: 'renamed')),
      expect: () => [
        isA<WorkspaceLoading>(),
        isA<WorkspaceLoaded>(),
        isA<WorkspaceOperationSuccess>().having((s) => s.message, 'message', 'Workspace updated'),
        isA<WorkspaceLoaded>().having((s) => s.workspaces.single.name, 'workspaces[0].name', 'renamed'),
      ],
    );

    blocTest<WorkspaceBloc, WorkspaceState>(
      'WorkspaceDeleteRequested success deletes then reloads',
      build: () {
        when(() => workspaceService.deleteWorkspace('ws-1')).thenAnswer((_) async {});
        when(() => workspaceService.getWorkspaces()).thenAnswer((_) async => const []);
        return WorkspaceBloc(workspaceService: workspaceService);
      },
      act: (bloc) => bloc.add(WorkspaceDeleteRequested('ws-1')),
      expect: () => [
        isA<WorkspaceLoading>(),
        isA<WorkspaceLoaded>().having((s) => s.workspaces, 'workspaces', isEmpty),
        isA<WorkspaceOperationSuccess>().having((s) => s.message, 'message', 'Workspace deleted'),
        isA<WorkspaceLoaded>().having((s) => s.workspaces, 'workspaces', isEmpty),
      ],
      verify: (_) {
        verify(() => workspaceService.deleteWorkspace('ws-1')).called(1);
      },
    );

    blocTest<WorkspaceBloc, WorkspaceState>(
      'WorkspaceAddMember success emits ActionSuccess then chains a real WorkspaceLoadRequested',
      build: () {
        when(() => workspaceService.addMember('ws-1', 'user-9', role: 'admin')).thenAnswer((_) async {});
        when(() => workspaceService.getWorkspaces()).thenAnswer((_) async => [workspace(id: 'ws-1')]);
        return WorkspaceBloc(workspaceService: workspaceService);
      },
      act: (bloc) => bloc.add(WorkspaceAddMember(workspaceId: 'ws-1', userId: 'user-9', role: 'admin')),
      expect: () => [
        isA<WorkspaceOperationSuccess>().having((s) => s.message, 'message', 'Member added'),
        isA<WorkspaceLoading>(),
        isA<WorkspaceLoaded>().having((s) => s.workspaces.single.id, 'workspaces[0].id', 'ws-1'),
      ],
      verify: (_) {
        verify(() => workspaceService.addMember('ws-1', 'user-9', role: 'admin')).called(1);
      },
    );

    blocTest<WorkspaceBloc, WorkspaceState>(
      'WorkspaceAddMember failure emits Error only, no member-add-triggered reload',
      build: () {
        when(() => workspaceService.addMember(any(), any(), role: any(named: 'role')))
            .thenThrow(WorkspaceServiceException('not a member'));
        return WorkspaceBloc(workspaceService: workspaceService);
      },
      act: (bloc) => bloc.add(WorkspaceAddMember(workspaceId: 'ws-1', userId: 'user-9')),
      expect: () => [isA<WorkspaceError>()],
      verify: (_) {
        verifyNever(() => workspaceService.getWorkspaces());
      },
    );
  });

  group('Workspace.copyWith (real model behaviour)', () {
    test('overrides only the requested fields and preserves the rest', () {
      final original = workspace(id: 'w-1', name: 'orig');
      final renamed = original.copyWith(name: 'renamed');

      expect(renamed.name, 'renamed');
      expect(renamed.id, original.id);
      expect(renamed.hostIds, original.hostIds);
    });
  });
}
