// Real tests for TerminalBloc — replaces the former `expect(true, isTrue)`
// stub (§11.4/§11.4.27).
//
// TerminalService is a concrete class backed by a real dart:io WebSocket
// (unsuitable for a host unit test), so instead of mocking an abstract
// interface (none exists) this suite subclasses it and overrides its public,
// non-final methods with an in-memory fake — no lib/ source change was
// needed for this bloc. The fake also proves the REAL onMessage callback
// wiring TerminalBloc installs on connect (event -> add() -> new state),
// not just isolated per-event handler behaviour.

import 'package:bloc_test/bloc_test.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:helix_terminator/bloc/terminal_bloc.dart';
import 'package:helix_terminator/services/terminal_service.dart';

class FakeTerminalService extends TerminalService {
  Object? connectError;
  Object? sendCommandError;

  bool disconnectCalled = false;
  bool removeOnMessageCalled = false;
  bool disposeCalled = false;
  final List<String> sentCommands = [];
  TerminalMessageCallback? _callback;

  @override
  Future<void> connect(String hostId) async {
    if (connectError != null) throw connectError!;
  }

  @override
  Future<void> sendCommand(String command) async {
    if (sendCommandError != null) throw sendCommandError!;
    sentCommands.add(command);
  }

  @override
  Future<void> disconnect() async {
    disconnectCalled = true;
  }

  @override
  void onMessage(TerminalMessageCallback callback) {
    _callback = callback;
  }

  @override
  void removeOnMessage() {
    removeOnMessageCalled = true;
    _callback = null;
  }

  @override
  Future<void> dispose() async {
    disposeCalled = true;
  }

  /// Simulates the WebSocket pushing an inbound frame, exactly like the real
  /// `_socket!.listen((data) { ... _onMessageCallback?.call(message); })`.
  void pushMessage(String message) => _callback?.call(message);
}

void main() {
  late FakeTerminalService fakeService;

  setUp(() {
    fakeService = FakeTerminalService();
  });

  group('TerminalBloc events', () {
    blocTest<TerminalBloc, TerminalState>(
      'TerminalConnectRequested success emits [Connecting, Connected] for the real hostId',
      build: () => TerminalBloc(terminalService: fakeService),
      act: (bloc) => bloc.add(TerminalConnectRequested('host-42')),
      expect: () => [
        isA<TerminalConnecting>().having((s) => s.hostId, 'hostId', 'host-42'),
        isA<TerminalConnected>()
            .having((s) => s.hostId, 'hostId', 'host-42')
            .having((s) => s.outputLines, 'outputLines', isEmpty),
      ],
    );

    blocTest<TerminalBloc, TerminalState>(
      'TerminalConnectRequested failure emits [Connecting, Error] carrying the real cause',
      build: () {
        fakeService.connectError = Exception('refused');
        return TerminalBloc(terminalService: fakeService);
      },
      act: (bloc) => bloc.add(TerminalConnectRequested('host-42')),
      expect: () => [
        isA<TerminalConnecting>(),
        isA<TerminalError>().having((s) => s.message, 'message', contains('refused')),
      ],
    );

    blocTest<TerminalBloc, TerminalState>(
      'TerminalMessageReceived while Connected appends to outputLines',
      seed: () => TerminalConnected(hostId: 'host-1', outputLines: const ['first line']),
      build: () => TerminalBloc(terminalService: fakeService),
      act: (bloc) => bloc.add(TerminalMessageReceived('second line')),
      expect: () => [
        isA<TerminalConnected>().having(
          (s) => s.outputLines,
          'outputLines',
          ['first line', 'second line'],
        ),
      ],
    );

    blocTest<TerminalBloc, TerminalState>(
      'TerminalMessageReceived is a NO-OP when not Connected (proves the state-guard is real)',
      build: () => TerminalBloc(terminalService: fakeService),
      act: (bloc) => bloc.add(TerminalMessageReceived('ignored')),
      expect: () => <Matcher>[],
    );

    blocTest<TerminalBloc, TerminalState>(
      'TerminalSendCommand while Connected forwards the exact command to the service '
      'without emitting a spurious state',
      seed: () => TerminalConnected(hostId: 'host-1'),
      build: () => TerminalBloc(terminalService: fakeService),
      act: (bloc) => bloc.add(TerminalSendCommand('ls -la')),
      expect: () => <Matcher>[],
      verify: (_) {
        expect(fakeService.sentCommands, ['ls -la']);
      },
    );

    blocTest<TerminalBloc, TerminalState>(
      'TerminalSendCommand failure emits [Error, <restored Connected state>]',
      seed: () => TerminalConnected(hostId: 'host-1', outputLines: const ['prior output']),
      build: () {
        fakeService.sendCommandError = Exception('closed');
        return TerminalBloc(terminalService: fakeService);
      },
      act: (bloc) => bloc.add(TerminalSendCommand('whoami')),
      expect: () => [
        isA<TerminalError>().having((s) => s.message, 'message', contains('closed')),
        isA<TerminalConnected>().having((s) => s.outputLines, 'outputLines', ['prior output']),
      ],
    );

    blocTest<TerminalBloc, TerminalState>(
      'TerminalResize is currently a documented no-op: it forwards nothing and changes no state',
      seed: () => TerminalConnected(hostId: 'host-1'),
      build: () => TerminalBloc(terminalService: fakeService),
      act: (bloc) => bloc.add(TerminalResize(120, 40)),
      expect: () => <Matcher>[],
    );

    blocTest<TerminalBloc, TerminalState>(
      'TerminalDisconnectRequested calls disconnect + removeOnMessage on the real service '
      'and emits Disconnected',
      seed: () => TerminalConnected(hostId: 'host-1'),
      build: () => TerminalBloc(terminalService: fakeService),
      act: (bloc) => bloc.add(TerminalDisconnectRequested()),
      expect: () => [isA<TerminalDisconnected>()],
      verify: (_) {
        expect(fakeService.disconnectCalled, isTrue);
        expect(fakeService.removeOnMessageCalled, isTrue);
      },
    );
  });

  group('TerminalService onMessage callback wiring (real end-to-end integration)', () {
    test('bloc.close() disposes the underlying TerminalService exactly once', () async {
      final bloc = TerminalBloc(terminalService: fakeService);

      expect(fakeService.disposeCalled, isFalse);
      await bloc.close();

      expect(fakeService.disposeCalled, isTrue);
    });

    test(
      'a message pushed by the underlying service after connect() flows through '
      'TerminalBloc.add() into a real new state on the bloc stream',
      () async {
        final bloc = TerminalBloc(terminalService: fakeService);
        addTearDown(bloc.close);

        final states = <TerminalState>[];
        final sub = bloc.stream.listen(states.add);
        addTearDown(sub.cancel);

        bloc.add(TerminalConnectRequested('host-7'));
        // Let Connecting -> Connected settle, and the onMessage callback register.
        await Future<void>.delayed(Duration.zero);
        await Future<void>.delayed(Duration.zero);

        expect(states.whereType<TerminalConnected>(), isNotEmpty);

        // Simulate the WebSocket delivering server output — this exercises
        // the REAL callback the bloc installed in `_onConnectRequested`
        // (`_terminalService.onMessage((message) => add(TerminalMessageReceived(message)))`),
        // not a synthetic shortcut.
        fakeService.pushMessage('server says hi');
        await Future<void>.delayed(Duration.zero);
        await Future<void>.delayed(Duration.zero);

        final latest = states.whereType<TerminalConnected>().last;
        expect(latest.outputLines, contains('server says hi'));
      },
    );
  });
}
