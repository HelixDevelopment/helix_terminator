// Real tests for [ApiClient] — replaces the former `expect(true, isTrue)`
// stub (§11.4/§11.4.27). Exercises the REAL HTTP request/response pipeline
// (method, path, headers, JSON body, status-code handling, auth-token
// injection) against `package:http`'s built-in `MockClient` — no live
// network call, but every byte of ApiClient's own logic runs for real.
//
// `httpClient` is an injectable seam added to ApiClient (lib/services/
// api_client.dart) purely for testability; production code is unaffected
// (it omits the parameter and gets a real http.Client() exactly as before).

import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:helix_terminator/services/api_client.dart';
import 'package:helix_terminator/services/auth_service.dart';

/// A minimal fake AuthService that returns a fixed token without touching
/// flutter_secure_storage (which has no platform implementation under
/// `flutter test`). AuthService's `getToken()` is a plain, overridable
/// instance method, so no source change was needed to make this possible.
class _FakeAuthService extends AuthService {
  _FakeAuthService(this._token) : super(apiClient: ApiClient(baseUrl: 'http://unused.invalid'));

  final String? _token;

  @override
  Future<String?> getToken() async => _token;
}

void main() {
  group('ApiClient.get', () {
    test('sends a GET to baseUrl+path and decodes a 200 JSON body', () async {
      Uri? capturedUri;
      String? capturedMethod;
      final mockClient = MockClient((request) async {
        capturedUri = request.url;
        capturedMethod = request.method;
        return http.Response(jsonEncode({'id': '42', 'name': 'demo'}), 200);
      });

      final client = ApiClient(baseUrl: 'https://api.example.test', httpClient: mockClient);
      final result = await client.get('/hosts/42');

      expect(capturedMethod, 'GET');
      expect(capturedUri, Uri.parse('https://api.example.test/hosts/42'));
      expect(result, {'id': '42', 'name': 'demo'});
    });

    test('an empty 200 body decodes to an empty map, not a crash', () async {
      final mockClient = MockClient((request) async => http.Response('', 200));
      final client = ApiClient(baseUrl: 'https://api.example.test', httpClient: mockClient);

      final result = await client.get('/hosts/42');

      expect(result, <String, dynamic>{});
    });

    test('always sends Content-Type: application/json', () async {
      Map<String, String>? capturedHeaders;
      final mockClient = MockClient((request) async {
        capturedHeaders = request.headers;
        return http.Response('{}', 200);
      });
      final client = ApiClient(baseUrl: 'https://api.example.test', httpClient: mockClient);

      await client.get('/anything');

      expect(capturedHeaders?['Content-Type'], 'application/json');
    });

    test('injects Authorization: Bearer <token> when the AuthService has a token', () async {
      Map<String, String>? capturedHeaders;
      final mockClient = MockClient((request) async {
        capturedHeaders = request.headers;
        return http.Response('{}', 200);
      });
      final client = ApiClient(
        baseUrl: 'https://api.example.test',
        httpClient: mockClient,
        authService: _FakeAuthService('secret-token-123'),
      );

      await client.get('/protected');

      expect(capturedHeaders?['Authorization'], 'Bearer secret-token-123');
    });

    test('sends no Authorization header when there is no token', () async {
      Map<String, String>? capturedHeaders;
      final mockClient = MockClient((request) async {
        capturedHeaders = request.headers;
        return http.Response('{}', 200);
      });
      final client = ApiClient(
        baseUrl: 'https://api.example.test',
        httpClient: mockClient,
        authService: _FakeAuthService(null),
      );

      await client.get('/public');

      expect(capturedHeaders?.containsKey('Authorization'), isFalse);
    });

    test('sends no Authorization header when no AuthService is configured at all', () async {
      Map<String, String>? capturedHeaders;
      final mockClient = MockClient((request) async {
        capturedHeaders = request.headers;
        return http.Response('{}', 200);
      });
      final client = ApiClient(baseUrl: 'https://api.example.test', httpClient: mockClient);

      await client.get('/public');

      expect(capturedHeaders?.containsKey('Authorization'), isFalse);
    });

    test('a non-2xx response throws ApiException carrying the parsed message + status code', () async {
      final mockClient = MockClient(
        (request) async => http.Response(jsonEncode({'message': 'Host not found'}), 404),
      );
      final client = ApiClient(baseUrl: 'https://api.example.test', httpClient: mockClient);

      await expectLater(
        client.get('/hosts/missing'),
        throwsA(
          isA<ApiException>()
              .having((e) => e.message, 'message', 'Host not found')
              .having((e) => e.statusCode, 'statusCode', 404),
        ),
      );
    });

    test('falls back to the "error" field when "message" is absent', () async {
      final mockClient = MockClient(
        (request) async => http.Response(jsonEncode({'error': 'boom'}), 500),
      );
      final client = ApiClient(baseUrl: 'https://api.example.test', httpClient: mockClient);

      await expectLater(
        client.get('/broken'),
        throwsA(isA<ApiException>().having((e) => e.message, 'message', 'boom')),
      );
    });
  });

  group('ApiClient.post', () {
    test('sends a JSON-encoded body to the right path and returns the decoded response', () async {
      String? capturedMethod;
      String? capturedBody;
      final mockClient = MockClient((request) async {
        capturedMethod = request.method;
        capturedBody = request.body;
        return http.Response(jsonEncode({'data': 'created'}), 201);
      });
      final client = ApiClient(baseUrl: 'https://api.example.test', httpClient: mockClient);

      final result = await client.post('/hosts', {'name': 'db-1', 'port': 22});

      expect(capturedMethod, 'POST');
      expect(jsonDecode(capturedBody!), {'name': 'db-1', 'port': 22});
      expect(result, {'data': 'created'});
    });
  });

  group('ApiClient.put', () {
    test('sends a PUT with the JSON body', () async {
      String? capturedMethod;
      final mockClient = MockClient((request) async {
        capturedMethod = request.method;
        return http.Response(jsonEncode({'data': 'updated'}), 200);
      });
      final client = ApiClient(baseUrl: 'https://api.example.test', httpClient: mockClient);

      final result = await client.put('/hosts/1', {'name': 'renamed'});

      expect(capturedMethod, 'PUT');
      expect(result, {'data': 'updated'});
    });
  });

  group('ApiClient.delete', () {
    test('sends a DELETE with no body and returns the decoded response', () async {
      Uri? capturedUri;
      String? capturedMethod;
      final mockClient = MockClient((request) async {
        capturedUri = request.url;
        capturedMethod = request.method;
        return http.Response('', 200);
      });
      final client = ApiClient(baseUrl: 'https://api.example.test', httpClient: mockClient);

      final result = await client.delete('/hosts/1');

      expect(capturedMethod, 'DELETE');
      expect(capturedUri, Uri.parse('https://api.example.test/hosts/1'));
      expect(result, <String, dynamic>{});
    });
  });
}
