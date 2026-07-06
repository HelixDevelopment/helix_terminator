import 'package:flutter_bloc/flutter_bloc.dart';
import 'api_client.dart';

// Events
abstract class ContainerBridgeEvent {}

class ContainerBridgeListRequested extends ContainerBridgeEvent {}

class ContainerBridgeCreate extends ContainerBridgeEvent {
  final String name;
  final String image;
  ContainerBridgeCreate({required this.name, required this.image});
}

class ContainerBridgeConnect extends ContainerBridgeEvent {
  final String id;
  ContainerBridgeConnect(this.id);
}

class ContainerBridgeStart extends ContainerBridgeEvent {
  final String id;
  ContainerBridgeStart(this.id);
}

class ContainerBridgeStop extends ContainerBridgeEvent {
  final String id;
  ContainerBridgeStop(this.id);
}

class ContainerBridgeRestart extends ContainerBridgeEvent {
  final String id;
  ContainerBridgeRestart(this.id);
}

class ContainerBridgeRemove extends ContainerBridgeEvent {
  final String id;
  ContainerBridgeRemove(this.id);
}

class ContainerBridgeLogs extends ContainerBridgeEvent {
  final String id;
  ContainerBridgeLogs(this.id);
}

// States
abstract class ContainerBridgeState {}

class ContainerBridgeInitial extends ContainerBridgeState {}

class ContainerBridgeLoading extends ContainerBridgeState {}

class ContainerBridgeListLoaded extends ContainerBridgeState {
  final List<Map<String, dynamic>> containers;
  ContainerBridgeListLoaded(this.containers);
}

class ContainerBridgeError extends ContainerBridgeState {
  final String message;
  ContainerBridgeError(this.message);
}

class ContainerBridgeActionSuccess extends ContainerBridgeState {
  final String message;
  ContainerBridgeActionSuccess(this.message);
}

// Bloc
class ContainerBridgeBloc extends Bloc<ContainerBridgeEvent, ContainerBridgeState> {
  final ApiClient _apiClient;

  ContainerBridgeBloc({required ApiClient apiClient})
      : _apiClient = apiClient,
        super(ContainerBridgeInitial()) {
    on<ContainerBridgeListRequested>(_onListRequested);
    on<ContainerBridgeCreate>(_onCreate);
    on<ContainerBridgeConnect>(_onConnect);
    on<ContainerBridgeStart>(_onStart);
    on<ContainerBridgeStop>(_onStop);
    on<ContainerBridgeRestart>(_onRestart);
    on<ContainerBridgeRemove>(_onRemove);
    on<ContainerBridgeLogs>(_onLogs);
  }

  Future<void> _onListRequested(ContainerBridgeListRequested event, Emitter<ContainerBridgeState> emit) async {
    emit(ContainerBridgeLoading());
    try {
      final response = await _apiClient.get('/api/v1/containers');
      final data = (response['data'] as List<dynamic>? ?? [])
          .map((e) => e as Map<String, dynamic>)
          .toList();
      emit(ContainerBridgeListLoaded(data));
    } catch (e) {
      emit(ContainerBridgeError(e.toString()));
    }
  }

  Future<void> _onCreate(ContainerBridgeCreate event, Emitter<ContainerBridgeState> emit) async {
    try {
      await _apiClient.post('/api/v1/containers', {
        'name': event.name,
        'image': event.image,
      });
      emit(ContainerBridgeActionSuccess('Container created'));
      add(ContainerBridgeListRequested());
    } catch (e) {
      emit(ContainerBridgeError(e.toString()));
    }
  }

  Future<void> _onConnect(ContainerBridgeConnect event, Emitter<ContainerBridgeState> emit) async {
    try {
      await _apiClient.post('/api/v1/containers/${event.id}/connect', {});
      emit(ContainerBridgeActionSuccess('Connected to container'));
    } catch (e) {
      emit(ContainerBridgeError(e.toString()));
    }
  }

  Future<void> _onStart(ContainerBridgeStart event, Emitter<ContainerBridgeState> emit) async {
    try {
      await _apiClient.post('/api/v1/containers/${event.id}/start', {});
      emit(ContainerBridgeActionSuccess('Container started'));
      add(ContainerBridgeListRequested());
    } catch (e) {
      emit(ContainerBridgeError(e.toString()));
    }
  }

  Future<void> _onStop(ContainerBridgeStop event, Emitter<ContainerBridgeState> emit) async {
    try {
      await _apiClient.post('/api/v1/containers/${event.id}/stop', {});
      emit(ContainerBridgeActionSuccess('Container stopped'));
      add(ContainerBridgeListRequested());
    } catch (e) {
      emit(ContainerBridgeError(e.toString()));
    }
  }

  Future<void> _onRestart(ContainerBridgeRestart event, Emitter<ContainerBridgeState> emit) async {
    try {
      await _apiClient.post('/api/v1/containers/${event.id}/restart', {});
      emit(ContainerBridgeActionSuccess('Container restarted'));
      add(ContainerBridgeListRequested());
    } catch (e) {
      emit(ContainerBridgeError(e.toString()));
    }
  }

  Future<void> _onRemove(ContainerBridgeRemove event, Emitter<ContainerBridgeState> emit) async {
    try {
      await _apiClient.post('/api/v1/containers/${event.id}/remove', {});
      emit(ContainerBridgeActionSuccess('Container removed'));
      add(ContainerBridgeListRequested());
    } catch (e) {
      emit(ContainerBridgeError(e.toString()));
    }
  }

  Future<void> _onLogs(ContainerBridgeLogs event, Emitter<ContainerBridgeState> emit) async {
    try {
      await _apiClient.get('/api/v1/containers/${event.id}/logs');
      emit(ContainerBridgeActionSuccess('Logs loaded'));
    } catch (e) {
      emit(ContainerBridgeError(e.toString()));
    }
  }
}
