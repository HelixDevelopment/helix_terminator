import 'package:flutter_bloc/flutter_bloc.dart';
import '../services/collaboration_service.dart';

// Events
abstract class CollaborationEvent {}

class CollaborationListRequested extends CollaborationEvent {}

class CollaborationCreateSession extends CollaborationEvent {
  final String hostId;
  final String? name;
  CollaborationCreateSession({required this.hostId, this.name});
}

class CollaborationJoinSession extends CollaborationEvent {
  final String sessionId;
  CollaborationJoinSession(this.sessionId);
}

class CollaborationLeaveSession extends CollaborationEvent {}

class CollaborationEndSession extends CollaborationEvent {
  final String sessionId;
  CollaborationEndSession(this.sessionId);
}

class CollaborationParticipantsRequested extends CollaborationEvent {
  final String sessionId;
  CollaborationParticipantsRequested(this.sessionId);
}

// States
abstract class CollaborationState {}

class CollaborationInitial extends CollaborationState {}

class CollaborationLoading extends CollaborationState {}

class CollaborationListLoaded extends CollaborationState {
  final List<dynamic> sessions;
  CollaborationListLoaded(this.sessions);
}

class CollaborationActive extends CollaborationState {
  final dynamic session;
  final List<Map<String, dynamic>> participants;
  CollaborationActive(this.session, {this.participants = const []});
}

class CollaborationError extends CollaborationState {
  final String message;
  CollaborationError(this.message);
}

class CollaborationActionSuccess extends CollaborationState {
  final String message;
  CollaborationActionSuccess(this.message);
}

// Bloc
class CollaborationBloc extends Bloc<CollaborationEvent, CollaborationState> {
  final CollaborationService _service;

  CollaborationBloc({required CollaborationService service})
      : _service = service,
        super(CollaborationInitial()) {
    on<CollaborationListRequested>(_onListRequested);
    on<CollaborationCreateSession>(_onCreateSession);
    on<CollaborationJoinSession>(_onJoinSession);
    on<CollaborationLeaveSession>(_onLeaveSession);
    on<CollaborationEndSession>(_onEndSession);
    on<CollaborationParticipantsRequested>(_onParticipantsRequested);
  }

  Future<void> _onListRequested(
    CollaborationListRequested event,
    Emitter<CollaborationState> emit,
  ) async {
    emit(CollaborationLoading());
    try {
      final sessions = await _service.getSessions();
      emit(CollaborationListLoaded(sessions));
    } catch (e) {
      emit(CollaborationError(e.toString()));
    }
  }

  Future<void> _onCreateSession(
    CollaborationCreateSession event,
    Emitter<CollaborationState> emit,
  ) async {
    try {
      final session = await _service.createSession(
        hostId: event.hostId,
        name: event.name,
      );
      emit(CollaborationActionSuccess('Session created'));
      emit(CollaborationActive(session));
    } catch (e) {
      emit(CollaborationError(e.toString()));
    }
  }

  Future<void> _onJoinSession(
    CollaborationJoinSession event,
    Emitter<CollaborationState> emit,
  ) async {
    try {
      final session = await _service.joinSession(event.sessionId);
      final participants = await _service.getParticipants(event.sessionId);
      emit(CollaborationActionSuccess('Joined session'));
      emit(CollaborationActive(session, participants: participants));
    } catch (e) {
      emit(CollaborationError(e.toString()));
    }
  }

  Future<void> _onLeaveSession(
    CollaborationLeaveSession event,
    Emitter<CollaborationState> emit,
  ) async {
    try {
      if (state is CollaborationActive) {
        final session = (state as CollaborationActive).session;
        await _service.leaveSession(session.id);
        emit(CollaborationActionSuccess('Left session'));
        add(CollaborationListRequested());
      }
    } catch (e) {
      emit(CollaborationError(e.toString()));
    }
  }

  Future<void> _onEndSession(
    CollaborationEndSession event,
    Emitter<CollaborationState> emit,
  ) async {
    try {
      await _service.endSession(event.sessionId);
      emit(CollaborationActionSuccess('Session ended'));
      add(CollaborationListRequested());
    } catch (e) {
      emit(CollaborationError(e.toString()));
    }
  }

  Future<void> _onParticipantsRequested(
    CollaborationParticipantsRequested event,
    Emitter<CollaborationState> emit,
  ) async {
    try {
      final participants = await _service.getParticipants(event.sessionId);
      if (state is CollaborationActive) {
        final current = state as CollaborationActive;
        emit(CollaborationActive(current.session, participants: participants));
      }
    } catch (e) {
      emit(CollaborationError(e.toString()));
    }
  }
}
