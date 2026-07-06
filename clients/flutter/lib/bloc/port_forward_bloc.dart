import 'package:flutter_bloc/flutter_bloc.dart';
import '../models/port_forward.dart';
import '../services/port_forward_service.dart';

// Events
abstract class PortForwardEvent {}

class PortForwardListRequested extends PortForwardEvent {}

class PortForwardCreate extends PortForwardEvent {
  final String hostId;
  final int localPort;
  final int remotePort;
  final String remoteHost;
  PortForwardCreate({
    required this.hostId,
    required this.localPort,
    required this.remotePort,
    this.remoteHost = 'localhost',
  });
}

class PortForwardDelete extends PortForwardEvent {
  final String id;
  PortForwardDelete(this.id);
}

class PortForwardStart extends PortForwardEvent {
  final String id;
  PortForwardStart(this.id);
}

class PortForwardStop extends PortForwardEvent {
  final String id;
  PortForwardStop(this.id);
}

class PortForwardActiveConnectionsRequested extends PortForwardEvent {}

// States
abstract class PortForwardState {}

class PortForwardInitial extends PortForwardState {}

class PortForwardLoading extends PortForwardState {}

class PortForwardListLoaded extends PortForwardState {
  final List<PortForward> rules;
  PortForwardListLoaded(this.rules);
}

class PortForwardActiveConnectionsLoaded extends PortForwardState {
  final List<Map<String, dynamic>> connections;
  PortForwardActiveConnectionsLoaded(this.connections);
}

class PortForwardError extends PortForwardState {
  final String message;
  PortForwardError(this.message);
}

class PortForwardActionSuccess extends PortForwardState {
  final String message;
  PortForwardActionSuccess(this.message);
}

// Bloc
class PortForwardBloc extends Bloc<PortForwardEvent, PortForwardState> {
  final PortForwardService _service;

  PortForwardBloc({required PortForwardService service})
      : _service = service,
        super(PortForwardInitial()) {
    on<PortForwardListRequested>(_onListRequested);
    on<PortForwardCreate>(_onCreate);
    on<PortForwardDelete>(_onDelete);
    on<PortForwardStart>(_onStart);
    on<PortForwardStop>(_onStop);
    on<PortForwardActiveConnectionsRequested>(_onActiveConnectionsRequested);
  }

  Future<void> _onListRequested(PortForwardListRequested event, Emitter<PortForwardState> emit) async {
    emit(PortForwardLoading());
    try {
      final rules = await _service.getPortForwards();
      emit(PortForwardListLoaded(rules));
    } catch (e) {
      emit(PortForwardError(e.toString()));
    }
  }

  Future<void> _onCreate(PortForwardCreate event, Emitter<PortForwardState> emit) async {
    try {
      await _service.createPortForward(
        hostId: event.hostId,
        localPort: event.localPort,
        remotePort: event.remotePort,
        remoteHost: event.remoteHost,
      );
      emit(PortForwardActionSuccess('Port forward created'));
      add(PortForwardListRequested());
    } catch (e) {
      emit(PortForwardError(e.toString()));
    }
  }

  Future<void> _onDelete(PortForwardDelete event, Emitter<PortForwardState> emit) async {
    try {
      await _service.deletePortForward(event.id);
      emit(PortForwardActionSuccess('Port forward deleted'));
      add(PortForwardListRequested());
    } catch (e) {
      emit(PortForwardError(e.toString()));
    }
  }

  Future<void> _onStart(PortForwardStart event, Emitter<PortForwardState> emit) async {
    try {
      await _service.startPortForward(event.id);
      emit(PortForwardActionSuccess('Port forward started'));
      add(PortForwardListRequested());
    } catch (e) {
      emit(PortForwardError(e.toString()));
    }
  }

  Future<void> _onStop(PortForwardStop event, Emitter<PortForwardState> emit) async {
    try {
      await _service.stopPortForward(event.id);
      emit(PortForwardActionSuccess('Port forward stopped'));
      add(PortForwardListRequested());
    } catch (e) {
      emit(PortForwardError(e.toString()));
    }
  }

  Future<void> _onActiveConnectionsRequested(PortForwardActiveConnectionsRequested event, Emitter<PortForwardState> emit) async {
    try {
      final connections = await _service.getActiveConnections();
      emit(PortForwardActiveConnectionsLoaded(connections));
    } catch (e) {
      emit(PortForwardError(e.toString()));
    }
  }
}
