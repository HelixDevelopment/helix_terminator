import 'package:flutter_bloc/flutter_bloc.dart';

import '../services/terminal_service.dart';

// ------------------------------------------------------------------
// Events
// ------------------------------------------------------------------

abstract class TerminalEvent {}

class TerminalConnectRequested extends TerminalEvent {
  final String hostId;
  TerminalConnectRequested(this.hostId);
}

class TerminalDisconnectRequested extends TerminalEvent {}

class TerminalSendCommand extends TerminalEvent {
  final String command;
  TerminalSendCommand(this.command);
}

class TerminalResize extends TerminalEvent {
  final int cols;
  final int rows;
  TerminalResize(this.cols, this.rows);
}

class TerminalMessageReceived extends TerminalEvent {
  final String message;
  TerminalMessageReceived(this.message);
}

// ------------------------------------------------------------------
// States
// ------------------------------------------------------------------

abstract class TerminalState {}

class TerminalInitial extends TerminalState {}

class TerminalConnecting extends TerminalState {
  final String hostId;
  TerminalConnecting(this.hostId);
}

class TerminalConnected extends TerminalState {
  final String hostId;
  final List<String> outputLines;
  TerminalConnected({required this.hostId, this.outputLines = const []});

  TerminalConnected copyWith({List<String>? outputLines}) {
    return TerminalConnected(
      hostId: hostId,
      outputLines: outputLines ?? this.outputLines,
    );
  }
}

class TerminalDisconnected extends TerminalState {}

class TerminalError extends TerminalState {
  final String message;
  TerminalError(this.message);
}

// ------------------------------------------------------------------
// BLoC
// ------------------------------------------------------------------

class TerminalBloc extends Bloc<TerminalEvent, TerminalState> {
  final TerminalService _terminalService;

  TerminalBloc({TerminalService? terminalService})
      : _terminalService = terminalService ?? TerminalService(),
        super(TerminalInitial()) {
    on<TerminalConnectRequested>(_onConnectRequested);
    on<TerminalDisconnectRequested>(_onDisconnectRequested);
    on<TerminalSendCommand>(_onSendCommand);
    on<TerminalResize>(_onResize);
    on<TerminalMessageReceived>(_onMessageReceived);
  }

  Future<void> _onConnectRequested(
    TerminalConnectRequested event,
    Emitter<TerminalState> emit,
  ) async {
    emit(TerminalConnecting(event.hostId));
    try {
      _terminalService.onMessage((message) {
        add(TerminalMessageReceived(message));
      });
      await _terminalService.connect(event.hostId);
      emit(TerminalConnected(hostId: event.hostId));
    } catch (e) {
      emit(TerminalError('Failed to connect: $e'));
    }
  }

  Future<void> _onDisconnectRequested(
    TerminalDisconnectRequested event,
    Emitter<TerminalState> emit,
  ) async {
    await _terminalService.disconnect();
    _terminalService.removeOnMessage();
    emit(TerminalDisconnected());
  }

  Future<void> _onSendCommand(
    TerminalSendCommand event,
    Emitter<TerminalState> emit,
  ) async {
    final currentState = state;
    if (currentState is TerminalConnected) {
      try {
        await _terminalService.sendCommand(event.command);
      } catch (e) {
        emit(TerminalError('Send failed: $e'));
        emit(currentState);
      }
    }
  }

  Future<void> _onResize(
    TerminalResize event,
    Emitter<TerminalState> emit,
  ) async {
    // Resize can be forwarded to the backend if the protocol supports it.
    // For now we keep the state unchanged.
  }

  Future<void> _onMessageReceived(
    TerminalMessageReceived event,
    Emitter<TerminalState> emit,
  ) async {
    final currentState = state;
    if (currentState is TerminalConnected) {
      final updated = List<String>.from(currentState.outputLines)
        ..add(event.message);
      // Keep a rolling buffer of the last 5000 lines to avoid unbounded growth.
      if (updated.length > 5000) {
        updated.removeRange(0, updated.length - 5000);
      }
      emit(currentState.copyWith(outputLines: updated));
    }
  }

  @override
  Future<void> close() {
    _terminalService.dispose();
    return super.close();
  }
}
