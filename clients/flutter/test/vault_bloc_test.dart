// Real tests for VaultBloc — replaces the former `expect(true, isTrue)` stub
// (§11.4/§11.4.27). Drives every real event through the real bloc against a
// mocked VaultService, asserting the REAL (slightly unusual) emit sequence
// each handler actually produces — including the double-Loaded emission
// pattern (Loaded -> OperationSuccess -> Loaded again) so a future
// regression that collapses/reorders it is caught, not silently accepted.

import 'package:bloc_test/bloc_test.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';
import 'package:helix_terminator/bloc/vault_bloc.dart';
import 'package:helix_terminator/models/secret.dart';
import 'package:helix_terminator/services/vault_service.dart';

class MockVaultService extends Mock implements VaultService {}

void main() {
  late MockVaultService vaultService;

  Secret secret({required String id, String name = 'api-key'}) => Secret(
        id: id,
        name: name,
        type: 'token',
        createdAt: DateTime.utc(2026, 1, 1),
      );

  setUp(() {
    vaultService = MockVaultService();
  });

  group('VaultBloc', () {
    blocTest<VaultBloc, VaultState>(
      'VaultLoadRequested success emits [Loading, Loaded] with the real secrets',
      build: () {
        when(() => vaultService.getSecrets()).thenAnswer((_) async => [secret(id: '1'), secret(id: '2')]);
        return VaultBloc(vaultService: vaultService);
      },
      act: (bloc) => bloc.add(VaultLoadRequested()),
      expect: () => [
        isA<VaultLoading>(),
        isA<VaultLoaded>().having((s) => s.secrets.map((e) => e.id), 'secret ids', ['1', '2']),
      ],
    );

    blocTest<VaultBloc, VaultState>(
      'VaultLoadRequested failure emits [Loading, Error] with a real wrapped message',
      build: () {
        when(() => vaultService.getSecrets()).thenThrow(VaultServiceException('vault sealed'));
        return VaultBloc(vaultService: vaultService);
      },
      act: (bloc) => bloc.add(VaultLoadRequested()),
      expect: () => [
        isA<VaultLoading>(),
        isA<VaultError>().having((s) => s.message, 'message', contains('vault sealed')),
      ],
    );

    blocTest<VaultBloc, VaultState>(
      'VaultCreateRequested success creates then reloads: '
      '[Loading, Loaded, OperationSuccess, Loaded] (the real 4-state sequence)',
      build: () {
        when(
          () => vaultService.createSecret(
            name: 'db-password',
            value: 'hunter2',
            type: 'password',
            category: 'infra',
            description: 'prod db',
          ),
        ).thenAnswer((_) async => secret(id: 'new-1', name: 'db-password'));
        when(() => vaultService.getSecrets()).thenAnswer((_) async => [secret(id: 'new-1', name: 'db-password')]);
        return VaultBloc(vaultService: vaultService);
      },
      act: (bloc) => bloc.add(
        VaultCreateRequested(
          name: 'db-password',
          value: 'hunter2',
          type: 'password',
          category: 'infra',
          description: 'prod db',
        ),
      ),
      expect: () => [
        isA<VaultLoading>(),
        isA<VaultLoaded>().having((s) => s.secrets.single.name, 'secrets[0].name', 'db-password'),
        isA<VaultOperationSuccess>().having((s) => s.message, 'message', 'Secret created'),
        isA<VaultLoaded>().having((s) => s.secrets.single.name, 'secrets[0].name', 'db-password'),
      ],
      verify: (_) {
        verify(
          () => vaultService.createSecret(
            name: 'db-password',
            value: 'hunter2',
            type: 'password',
            category: 'infra',
            description: 'prod db',
          ),
        ).called(1);
      },
    );

    blocTest<VaultBloc, VaultState>(
      'VaultCreateRequested failure emits [Loading, Error] and never reloads the list',
      build: () {
        when(
          () => vaultService.createSecret(
            name: any(named: 'name'),
            value: any(named: 'value'),
            type: any(named: 'type'),
            category: any(named: 'category'),
            description: any(named: 'description'),
          ),
        ).thenThrow(VaultServiceException('duplicate name'));
        return VaultBloc(vaultService: vaultService);
      },
      act: (bloc) => bloc.add(VaultCreateRequested(name: 'dup', value: 'v', type: 'token')),
      expect: () => [
        isA<VaultLoading>(),
        isA<VaultError>().having((s) => s.message, 'message', contains('duplicate name')),
      ],
      verify: (_) {
        verifyNever(() => vaultService.getSecrets());
      },
    );

    blocTest<VaultBloc, VaultState>(
      'VaultUpdateRequested success updates then reloads the real 4-state sequence',
      build: () {
        when(
          () => vaultService.updateSecret(
            'sec-1',
            name: 'renamed',
            value: null,
            type: null,
            category: null,
            description: null,
          ),
        ).thenAnswer((_) async => secret(id: 'sec-1', name: 'renamed'));
        when(() => vaultService.getSecrets()).thenAnswer((_) async => [secret(id: 'sec-1', name: 'renamed')]);
        return VaultBloc(vaultService: vaultService);
      },
      act: (bloc) => bloc.add(VaultUpdateRequested(id: 'sec-1', name: 'renamed')),
      expect: () => [
        isA<VaultLoading>(),
        isA<VaultLoaded>(),
        isA<VaultOperationSuccess>().having((s) => s.message, 'message', 'Secret updated'),
        isA<VaultLoaded>().having((s) => s.secrets.single.name, 'secrets[0].name', 'renamed'),
      ],
    );

    blocTest<VaultBloc, VaultState>(
      'VaultDeleteRequested success deletes then reloads, real 4-state sequence',
      build: () {
        when(() => vaultService.deleteSecret('sec-1')).thenAnswer((_) async {});
        when(() => vaultService.getSecrets()).thenAnswer((_) async => const []);
        return VaultBloc(vaultService: vaultService);
      },
      act: (bloc) => bloc.add(VaultDeleteRequested('sec-1')),
      expect: () => [
        isA<VaultLoading>(),
        isA<VaultLoaded>().having((s) => s.secrets, 'secrets', isEmpty),
        isA<VaultOperationSuccess>().having((s) => s.message, 'message', 'Secret deleted'),
        isA<VaultLoaded>().having((s) => s.secrets, 'secrets', isEmpty),
      ],
      verify: (_) {
        verify(() => vaultService.deleteSecret('sec-1')).called(1);
      },
    );

    blocTest<VaultBloc, VaultState>(
      'VaultDeleteRequested failure emits [Loading, Error] and never calls getSecrets again',
      build: () {
        when(() => vaultService.deleteSecret('missing')).thenThrow(VaultServiceException('not found'));
        return VaultBloc(vaultService: vaultService);
      },
      act: (bloc) => bloc.add(VaultDeleteRequested('missing')),
      expect: () => [
        isA<VaultLoading>(),
        isA<VaultError>().having((s) => s.message, 'message', contains('not found')),
      ],
      verify: (_) {
        verifyNever(() => vaultService.getSecrets());
      },
    );
  });

  group('Secret.copyWith (real model behaviour)', () {
    test('overrides only the requested fields and preserves the rest', () {
      final original = secret(id: 's-1', name: 'orig');
      final updated = original.copyWith(name: 'renamed', category: 'infra');

      expect(updated.name, 'renamed');
      expect(updated.category, 'infra');
      expect(updated.id, original.id);
      expect(updated.type, original.type);
    });
  });
}
