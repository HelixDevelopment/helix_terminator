// Real tests for HostBloc — replaces the former `expect(true, isTrue)` stub
// (§11.4/§11.4.27). Drives every real event through the real bloc against a
// mocked HostService, asserting the real emitted HostState sequence
// including the previousHosts-preserving loading states and the
// create/update/delete list-mutation logic.
//
// Assertions use isA<T>().having(...) property matchers rather than direct
// `==` on HostState, because HostLoaded/HostError/HostOperationSuccess
// compare their `List<Host>` fields with plain `==` (List has no built-in
// structural equality), which would make identical-looking-but-different
// list instances spuriously unequal. `having(..., equals(...))` performs a
// real structural comparison instead, so these tests actually catch bugs.

import 'package:bloc_test/bloc_test.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';
import 'package:helix_terminator/bloc/host_bloc.dart';
import 'package:helix_terminator/models/host.dart';
import 'package:helix_terminator/services/host_service.dart';

class MockHostService extends Mock implements HostService {}

void main() {
  late MockHostService hostService;

  Host host({required String id, String name = 'db-1', String status = 'online'}) => Host(
        id: id,
        name: name,
        address: '10.0.0.$id',
        createdAt: DateTime.utc(2026, 1, 1),
        status: status,
      );

  setUpAll(() {
    // mocktail requires a registered fallback value for any() used with a
    // non-builtin argument type (Host, below), so it has a typed stand-in
    // for its internal invocation matching.
    registerFallbackValue(Host(id: 'fallback', name: 'fallback', address: '0.0.0.0', createdAt: DateTime.utc(2000)));
  });

  setUp(() {
    hostService = MockHostService();
  });

  group('HostBloc', () {
    blocTest<HostBloc, HostState>(
      'HostLoadRequested success emits [Loading(previousHosts: null), Loaded(hosts)]',
      build: () {
        when(() => hostService.getHosts()).thenAnswer((_) async => [host(id: '1'), host(id: '2')]);
        return HostBloc(hostService: hostService);
      },
      act: (bloc) => bloc.add(const HostLoadRequested()),
      expect: () => [
        isA<HostLoading>().having((s) => s.previousHosts, 'previousHosts', isNull),
        isA<HostLoaded>().having((s) => s.hosts.map((h) => h.id), 'hosts ids', ['1', '2']),
      ],
    );

    blocTest<HostBloc, HostState>(
      'HostLoadRequested failure emits [Loading, Error] carrying the real error text',
      build: () {
        when(() => hostService.getHosts()).thenThrow(HostServiceException('Unauthorized'));
        return HostBloc(hostService: hostService);
      },
      act: (bloc) => bloc.add(const HostLoadRequested()),
      expect: () => [
        isA<HostLoading>(),
        isA<HostError>().having((s) => s.message, 'message', contains('Unauthorized')),
      ],
    );

    blocTest<HostBloc, HostState>(
      'HostRefreshRequested while already Loaded preserves the previous hosts on the Loading state',
      seed: () => HostLoaded([host(id: '1')]),
      build: () {
        when(() => hostService.getHosts()).thenAnswer((_) async => [host(id: '1'), host(id: '2')]);
        return HostBloc(hostService: hostService);
      },
      act: (bloc) => bloc.add(const HostRefreshRequested()),
      expect: () => [
        isA<HostLoading>().having(
          (s) => s.previousHosts?.map((h) => h.id).toList(),
          'previousHosts ids',
          ['1'],
        ),
        isA<HostLoaded>().having((s) => s.hosts.length, 'hosts.length', 2),
      ],
    );

    blocTest<HostBloc, HostState>(
      'HostCreateRequested success appends the created host to the existing list',
      seed: () => HostLoaded([host(id: '1')]),
      build: () {
        when(() => hostService.createHost(any())).thenAnswer((_) async => host(id: '2', name: 'new-box'));
        return HostBloc(hostService: hostService);
      },
      act: (bloc) => bloc.add(HostCreateRequested(host(id: '2', name: 'new-box'))),
      expect: () => [
        isA<HostLoading>(),
        isA<HostOperationSuccess>()
            .having((s) => s.message, 'message', 'Host created successfully')
            .having((s) => s.hosts.map((h) => h.id).toList(), 'hosts ids', ['1', '2']),
        isA<HostLoaded>().having((s) => s.hosts.map((h) => h.id).toList(), 'hosts ids', ['1', '2']),
      ],
    );

    blocTest<HostBloc, HostState>(
      'HostCreateRequested failure keeps the pre-existing hosts as previousHosts on the Error state',
      seed: () => HostLoaded([host(id: '1')]),
      build: () {
        when(() => hostService.createHost(any())).thenThrow(HostServiceException('name taken'));
        return HostBloc(hostService: hostService);
      },
      act: (bloc) => bloc.add(HostCreateRequested(host(id: '2'))),
      expect: () => [
        isA<HostLoading>(),
        isA<HostError>()
            .having((s) => s.message, 'message', contains('name taken'))
            .having((s) => s.previousHosts?.map((h) => h.id).toList(), 'previousHosts ids', ['1']),
      ],
    );

    blocTest<HostBloc, HostState>(
      'HostUpdateRequested replaces exactly the matching host by id, leaving others untouched',
      seed: () => HostLoaded([host(id: '1', name: 'old-name'), host(id: '2', name: 'kept')]),
      build: () {
        when(() => hostService.updateHost('1', any())).thenAnswer((_) async => host(id: '1', name: 'renamed'));
        return HostBloc(hostService: hostService);
      },
      act: (bloc) => bloc.add(HostUpdateRequested('1', host(id: '1', name: 'renamed'))),
      expect: () => [
        isA<HostLoading>(),
        isA<HostOperationSuccess>(),
        isA<HostLoaded>().having(
          (s) => {for (final h in s.hosts) h.id: h.name},
          'id->name map',
          {'1': 'renamed', '2': 'kept'},
        ),
      ],
    );

    blocTest<HostBloc, HostState>(
      'HostDeleteRequested removes exactly the deleted host id from the list',
      seed: () => HostLoaded([host(id: '1'), host(id: '2'), host(id: '3')]),
      build: () {
        when(() => hostService.deleteHost('2')).thenAnswer((_) async {});
        return HostBloc(hostService: hostService);
      },
      act: (bloc) => bloc.add(const HostDeleteRequested('2')),
      expect: () => [
        isA<HostLoading>(),
        isA<HostOperationSuccess>().having((s) => s.hosts.map((h) => h.id).toList(), 'hosts ids', ['1', '3']),
        isA<HostLoaded>().having((s) => s.hosts.map((h) => h.id).toList(), 'hosts ids', ['1', '3']),
      ],
      verify: (_) {
        verify(() => hostService.deleteHost('2')).called(1);
      },
    );
  });

  group('Host model equality (used by HostBloc list-diffing above)', () {
    test('two Hosts with identical field values are equal; a changed field breaks equality', () {
      final a = host(id: '1', name: 'same');
      final b = host(id: '1', name: 'same');
      final different = host(id: '1', name: 'different');

      expect(a, equals(b));
      expect(a == different, isFalse);
    });

    test('copyWith overrides only the requested fields', () {
      final original = host(id: '1', name: 'orig', status: 'online');
      final renamed = original.copyWith(name: 'renamed');

      expect(renamed.name, 'renamed');
      expect(renamed.id, original.id);
      expect(renamed.status, original.status);
    });
  });
}
