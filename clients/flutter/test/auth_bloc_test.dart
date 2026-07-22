// Real tests for AuthBloc — replaces the former `expect(true, isTrue)` stub
// (§11.4/§11.4.27). Drives every real event through the real bloc against a
// mocked AuthService, asserting the real emitted AuthState sequence for the
// success, 2FA, and failure paths. Also proves the User model's
// fromJson/toJson round-trip is real (not a tautology).

import 'package:bloc_test/bloc_test.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';
import 'package:helix_terminator/bloc/auth_bloc.dart';
import 'package:helix_terminator/models/user.dart';
import 'package:helix_terminator/services/auth_service.dart';

class MockAuthService extends Mock implements AuthService {}

void main() {
  late MockAuthService authService;

  final user = User(
    id: 'user-1',
    email: 'demo@example.test',
    name: 'Demo User',
    createdAt: DateTime.utc(2026, 1, 1, 12, 0, 0),
  );

  setUp(() {
    authService = MockAuthService();
  });

  group('User model round-trip (real de/serialization, not a tautology)', () {
    test('toJson() then fromJson() reproduces every field', () {
      final withAvatar = User(
        id: 'u-9',
        email: 'round@trip.test',
        name: 'Round Trip',
        avatarUrl: 'https://cdn.example.test/a.png',
        createdAt: DateTime.utc(2026, 3, 4, 5, 6, 7),
      );

      final json = withAvatar.toJson();
      final decoded = User.fromJson(json);

      expect(decoded.id, withAvatar.id);
      expect(decoded.email, withAvatar.email);
      expect(decoded.name, withAvatar.name);
      expect(decoded.avatarUrl, withAvatar.avatarUrl);
      expect(decoded.createdAt, withAvatar.createdAt);
      // Real proof the JSON wire-format keys are exactly what the backend
      // contract expects (snake_case), not just that the round-trip agrees
      // with itself.
      expect(json['avatar_url'], 'https://cdn.example.test/a.png');
      expect(json['created_at'], '2026-03-04T05:06:07.000Z');
    });

    test('fromJson() tolerates a null avatar_url (optional field)', () {
      final decoded = User.fromJson({
        'id': 'u-2',
        'email': 'no-avatar@example.test',
        'name': 'No Avatar',
        'avatar_url': null,
        'created_at': '2026-01-01T00:00:00.000Z',
      });

      expect(decoded.avatarUrl, isNull);
      expect(decoded.email, 'no-avatar@example.test');
    });
  });

  group('AuthBloc', () {
    blocTest<AuthBloc, AuthState>(
      'AuthLoginRequested with valid credentials (no 2FA) emits [Loading, Authenticated]',
      build: () {
        when(() => authService.login('demo@example.test', 'correct-horse'))
            .thenAnswer((_) async => AuthResult(user: user));
        return AuthBloc(authService: authService);
      },
      act: (bloc) => bloc.add(
        AuthLoginRequested(email: 'demo@example.test', password: 'correct-horse'),
      ),
      expect: () => [
        isA<AuthLoading>(),
        isA<AuthAuthenticated>().having((s) => s.user.id, 'user.id', 'user-1'),
      ],
      verify: (_) {
        verify(() => authService.login('demo@example.test', 'correct-horse')).called(1);
      },
    );

    blocTest<AuthBloc, AuthState>(
      'AuthLoginRequested that requires 2FA emits [Loading, Auth2FARequired] — never Authenticated',
      build: () {
        when(() => authService.login('demo@example.test', 'correct-horse')).thenAnswer(
          (_) async => AuthResult(requires2FA: true, tempToken: 'temp-abc'),
        );
        return AuthBloc(authService: authService);
      },
      act: (bloc) => bloc.add(
        AuthLoginRequested(email: 'demo@example.test', password: 'correct-horse'),
      ),
      expect: () => [
        isA<AuthLoading>(),
        isA<Auth2FARequired>().having((s) => s.tempToken, 'tempToken', 'temp-abc'),
      ],
    );

    blocTest<AuthBloc, AuthState>(
      'AuthLoginRequested with bad credentials emits [Loading, Error] carrying the service message',
      build: () {
        when(() => authService.login('demo@example.test', 'wrong'))
            .thenThrow(AuthException('Invalid email or password.'));
        return AuthBloc(authService: authService);
      },
      act: (bloc) => bloc.add(AuthLoginRequested(email: 'demo@example.test', password: 'wrong')),
      expect: () => [
        isA<AuthLoading>(),
        isA<AuthError>().having((s) => s.message, 'message', 'Invalid email or password.'),
      ],
    );

    blocTest<AuthBloc, AuthState>(
      'AuthLoginRequested that throws a non-AuthException still surfaces a friendly AuthError '
      '(no unhandled exception escapes the bloc)',
      build: () {
        when(() => authService.login(any(), any())).thenThrow(Exception('socket reset'));
        return AuthBloc(authService: authService);
      },
      act: (bloc) => bloc.add(AuthLoginRequested(email: 'x@x.test', password: 'y')),
      expect: () => [
        isA<AuthLoading>(),
        isA<AuthError>().having(
          (s) => s.message,
          'message',
          'An unexpected error occurred. Please try again.',
        ),
      ],
    );

    blocTest<AuthBloc, AuthState>(
      'AuthCheckRequested with a valid session emits [Loading, Authenticated]',
      build: () {
        when(() => authService.isAuthenticated()).thenAnswer((_) async => true);
        when(() => authService.getCurrentUser()).thenAnswer((_) async => user);
        return AuthBloc(authService: authService);
      },
      act: (bloc) => bloc.add(AuthCheckRequested()),
      expect: () => [
        isA<AuthLoading>(),
        isA<AuthAuthenticated>().having((s) => s.user.email, 'user.email', user.email),
      ],
    );

    blocTest<AuthBloc, AuthState>(
      'AuthCheckRequested with no stored session emits [Loading, Unauthenticated]',
      build: () {
        when(() => authService.isAuthenticated()).thenAnswer((_) async => false);
        return AuthBloc(authService: authService);
      },
      act: (bloc) => bloc.add(AuthCheckRequested()),
      expect: () => [isA<AuthLoading>(), isA<AuthUnauthenticated>()],
      verify: (_) {
        // getCurrentUser must NEVER be called when there is no session —
        // proves the short-circuit branch really short-circuits.
        verifyNever(() => authService.getCurrentUser());
      },
    );

    blocTest<AuthBloc, AuthState>(
      'AuthCheckRequested that throws (e.g. secure-storage failure) still resolves to '
      'Unauthenticated, never leaves the app stuck on Loading',
      build: () {
        when(() => authService.isAuthenticated()).thenThrow(Exception('storage unavailable'));
        return AuthBloc(authService: authService);
      },
      act: (bloc) => bloc.add(AuthCheckRequested()),
      expect: () => [isA<AuthLoading>(), isA<AuthUnauthenticated>()],
    );

    blocTest<AuthBloc, AuthState>(
      'AuthLogoutRequested emits [Loading, Unauthenticated] and really calls the service',
      build: () {
        when(() => authService.logout()).thenAnswer((_) async {});
        return AuthBloc(authService: authService);
      },
      act: (bloc) => bloc.add(AuthLogoutRequested()),
      expect: () => [isA<AuthLoading>(), isA<AuthUnauthenticated>()],
      verify: (_) {
        verify(() => authService.logout()).called(1);
      },
    );

    blocTest<AuthBloc, AuthState>(
      'AuthRegisterRequested success emits [Loading, Authenticated] with the created user',
      build: () {
        when(
          () => authService.register(
            'new@example.test',
            'pw123456',
            'New Person',
            organizationName: 'Acme',
          ),
        ).thenAnswer((_) async => user);
        return AuthBloc(authService: authService);
      },
      act: (bloc) => bloc.add(
        AuthRegisterRequested(
          email: 'new@example.test',
          password: 'pw123456',
          name: 'New Person',
          organizationName: 'Acme',
        ),
      ),
      expect: () => [
        isA<AuthLoading>(),
        isA<AuthAuthenticated>().having((s) => s.user.id, 'user.id', user.id),
      ],
    );

    blocTest<AuthBloc, AuthState>(
      'Auth2FAVerified with a correct code emits [Loading, Authenticated]',
      build: () {
        when(() => authService.verify2FA('123456')).thenAnswer((_) async => user);
        return AuthBloc(authService: authService);
      },
      act: (bloc) => bloc.add(Auth2FAVerified(code: '123456')),
      expect: () => [isA<AuthLoading>(), isA<AuthAuthenticated>()],
    );

    blocTest<AuthBloc, AuthState>(
      'Auth2FAVerified with a wrong code emits [Loading, Error]',
      build: () {
        when(() => authService.verify2FA('000000')).thenThrow(AuthException('Invalid code.'));
        return AuthBloc(authService: authService);
      },
      act: (bloc) => bloc.add(Auth2FAVerified(code: '000000')),
      expect: () => [
        isA<AuthLoading>(),
        isA<AuthError>().having((s) => s.message, 'message', 'Invalid code.'),
      ],
    );
  });
}
