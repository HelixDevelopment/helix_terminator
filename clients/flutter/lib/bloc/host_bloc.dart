import 'package:flutter_bloc/flutter_bloc.dart';
import '../models/host.dart';
import '../services/host_service.dart';

// Events
abstract class HostEvent {
  const HostEvent();
}

class HostLoadRequested extends HostEvent {
  const HostLoadRequested();
}

class HostRefreshRequested extends HostEvent {
  const HostRefreshRequested();
}

class HostCreateRequested extends HostEvent {
  final Host host;
  const HostCreateRequested(this.host);
}

class HostUpdateRequested extends HostEvent {
  final String id;
  final Host host;
  const HostUpdateRequested(this.id, this.host);
}

class HostDeleteRequested extends HostEvent {
  final String id;
  const HostDeleteRequested(this.id);
}

// States
abstract class HostState {
  const HostState();

  @override
  bool operator ==(Object other) =>
      identical(this, other) || other is HostState && runtimeType == other.runtimeType;

  @override
  int get hashCode => runtimeType.hashCode;
}

class HostInitial extends HostState {
  const HostInitial();
}

class HostLoading extends HostState {
  final List<Host>? previousHosts;
  const HostLoading({this.previousHosts});

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is HostLoading &&
          runtimeType == other.runtimeType &&
          previousHosts == other.previousHosts;

  @override
  int get hashCode => previousHosts.hashCode;
}

class HostLoaded extends HostState {
  final List<Host> hosts;
  const HostLoaded(this.hosts);

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is HostLoaded &&
          runtimeType == other.runtimeType &&
          hosts == other.hosts;

  @override
  int get hashCode => hosts.hashCode;
}

class HostError extends HostState {
  final String message;
  final List<Host>? previousHosts;
  const HostError(this.message, {this.previousHosts});

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is HostError &&
          runtimeType == other.runtimeType &&
          message == other.message &&
          previousHosts == other.previousHosts;

  @override
  int get hashCode => message.hashCode ^ previousHosts.hashCode;
}

class HostOperationSuccess extends HostState {
  final String message;
  final List<Host> hosts;
  const HostOperationSuccess(this.message, this.hosts);

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is HostOperationSuccess &&
          runtimeType == other.runtimeType &&
          message == other.message &&
          hosts == other.hosts;

  @override
  int get hashCode => message.hashCode ^ hosts.hashCode;
}

// BLoC
class HostBloc extends Bloc<HostEvent, HostState> {
  final HostService _hostService;

  HostBloc({required HostService hostService})
      : _hostService = hostService,
        super(const HostInitial()) {
    on<HostLoadRequested>(_onLoadRequested);
    on<HostRefreshRequested>(_onRefreshRequested);
    on<HostCreateRequested>(_onCreateRequested);
    on<HostUpdateRequested>(_onUpdateRequested);
    on<HostDeleteRequested>(_onDeleteRequested);
  }

  Future<void> _onLoadRequested(
    HostLoadRequested event,
    Emitter<HostState> emit,
  ) async {
    final currentHosts = state is HostLoaded ? (state as HostLoaded).hosts : null;
    emit(HostLoading(previousHosts: currentHosts));
    try {
      final hosts = await _hostService.getHosts();
      emit(HostLoaded(hosts));
    } catch (e) {
      emit(HostError(e.toString(), previousHosts: currentHosts));
    }
  }

  Future<void> _onRefreshRequested(
    HostRefreshRequested event,
    Emitter<HostState> emit,
  ) async {
    final currentHosts = state is HostLoaded ? (state as HostLoaded).hosts : null;
    emit(HostLoading(previousHosts: currentHosts));
    try {
      final hosts = await _hostService.getHosts();
      emit(HostLoaded(hosts));
    } catch (e) {
      emit(HostError(e.toString(), previousHosts: currentHosts));
    }
  }

  Future<void> _onCreateRequested(
    HostCreateRequested event,
    Emitter<HostState> emit,
  ) async {
    final currentHosts = state is HostLoaded ? (state as HostLoaded).hosts : <Host>[];
    emit(HostLoading(previousHosts: currentHosts));
    try {
      final created = await _hostService.createHost(event.host);
      final updatedHosts = [...currentHosts, created];
      emit(HostOperationSuccess('Host created successfully', updatedHosts));
      emit(HostLoaded(updatedHosts));
    } catch (e) {
      emit(HostError(e.toString(), previousHosts: currentHosts));
    }
  }

  Future<void> _onUpdateRequested(
    HostUpdateRequested event,
    Emitter<HostState> emit,
  ) async {
    final currentHosts = state is HostLoaded ? (state as HostLoaded).hosts : <Host>[];
    emit(HostLoading(previousHosts: currentHosts));
    try {
      final updated = await _hostService.updateHost(event.id, event.host);
      final updatedHosts = currentHosts.map((h) => h.id == event.id ? updated : h).toList();
      emit(HostOperationSuccess('Host updated successfully', updatedHosts));
      emit(HostLoaded(updatedHosts));
    } catch (e) {
      emit(HostError(e.toString(), previousHosts: currentHosts));
    }
  }

  Future<void> _onDeleteRequested(
    HostDeleteRequested event,
    Emitter<HostState> emit,
  ) async {
    final currentHosts = state is HostLoaded ? (state as HostLoaded).hosts : <Host>[];
    emit(HostLoading(previousHosts: currentHosts));
    try {
      await _hostService.deleteHost(event.id);
      final updatedHosts = currentHosts.where((h) => h.id != event.id).toList();
      emit(HostOperationSuccess('Host deleted successfully', updatedHosts));
      emit(HostLoaded(updatedHosts));
    } catch (e) {
      emit(HostError(e.toString(), previousHosts: currentHosts));
    }
  }
}
