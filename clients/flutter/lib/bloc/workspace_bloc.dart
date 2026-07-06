import 'package:flutter_bloc/flutter_bloc.dart';

import '../models/workspace.dart';
import '../services/workspace_service.dart';

// ------------------------------------------------------------------
// Events
// ------------------------------------------------------------------

abstract class WorkspaceEvent {}

class WorkspaceLoadRequested extends WorkspaceEvent {}

class WorkspaceCreateRequested extends WorkspaceEvent {
  final String name;
  final String? description;
  final List<String> hostIds;
  WorkspaceCreateRequested({
    required this.name,
    this.description,
    this.hostIds = const [],
  });
}

class WorkspaceUpdateRequested extends WorkspaceEvent {
  final String id;
  final String? name;
  final String? description;
  final List<String>? hostIds;
  WorkspaceUpdateRequested({
    required this.id,
    this.name,
    this.description,
    this.hostIds,
  });
}

class WorkspaceDeleteRequested extends WorkspaceEvent {
  final String id;
  WorkspaceDeleteRequested(this.id);
}

class WorkspaceAddMember extends WorkspaceEvent {
  final String workspaceId;
  final String userId;
  final String role;
  WorkspaceAddMember({
    required this.workspaceId,
    required this.userId,
    this.role = 'member',
  });
}

// ------------------------------------------------------------------
// States
// ------------------------------------------------------------------

abstract class WorkspaceState {}

class WorkspaceInitial extends WorkspaceState {}

class WorkspaceLoading extends WorkspaceState {}

class WorkspaceLoaded extends WorkspaceState {
  final List<Workspace> workspaces;
  WorkspaceLoaded(this.workspaces);
}

class WorkspaceOperationSuccess extends WorkspaceState {
  final String message;
  WorkspaceOperationSuccess(this.message);
}

class WorkspaceError extends WorkspaceState {
  final String message;
  WorkspaceError(this.message);
}

// ------------------------------------------------------------------
// BLoC
// ------------------------------------------------------------------

class WorkspaceBloc extends Bloc<WorkspaceEvent, WorkspaceState> {
  final WorkspaceService _workspaceService;

  WorkspaceBloc({WorkspaceService? workspaceService})
      : _workspaceService = workspaceService ?? WorkspaceService(),
        super(WorkspaceInitial()) {
    on<WorkspaceLoadRequested>(_onLoad);
    on<WorkspaceCreateRequested>(_onCreate);
    on<WorkspaceUpdateRequested>(_onUpdate);
    on<WorkspaceDeleteRequested>(_onDelete);
    on<WorkspaceAddMember>(_onAddMember);
  }

  Future<void> _onLoad(
    WorkspaceLoadRequested event,
    Emitter<WorkspaceState> emit,
  ) async {
    emit(WorkspaceLoading());
    try {
      final workspaces = await _workspaceService.getWorkspaces();
      emit(WorkspaceLoaded(workspaces));
    } catch (e) {
      emit(WorkspaceError('Failed to load workspaces: $e'));
    }
  }

  Future<void> _onCreate(
    WorkspaceCreateRequested event,
    Emitter<WorkspaceState> emit,
  ) async {
    final previousState = state;
    emit(WorkspaceLoading());
    try {
      await _workspaceService.createWorkspace(
        name: event.name,
        description: event.description,
        hostIds: event.hostIds,
      );
      final workspaces = await _workspaceService.getWorkspaces();
      emit(WorkspaceLoaded(workspaces));
      emit(WorkspaceOperationSuccess('Workspace created'));
      // Restore loaded data so UI stays consistent.
      if (previousState is WorkspaceLoaded) {
        emit(WorkspaceLoaded(workspaces));
      }
    } catch (e) {
      emit(WorkspaceError('Failed to create workspace: $e'));
    }
  }

  Future<void> _onUpdate(
    WorkspaceUpdateRequested event,
    Emitter<WorkspaceState> emit,
  ) async {
    emit(WorkspaceLoading());
    try {
      await _workspaceService.updateWorkspace(
        event.id,
        name: event.name,
        description: event.description,
        hostIds: event.hostIds,
      );
      final workspaces = await _workspaceService.getWorkspaces();
      emit(WorkspaceLoaded(workspaces));
      emit(WorkspaceOperationSuccess('Workspace updated'));
      emit(WorkspaceLoaded(workspaces));
    } catch (e) {
      emit(WorkspaceError('Failed to update workspace: $e'));
    }
  }

  Future<void> _onDelete(
    WorkspaceDeleteRequested event,
    Emitter<WorkspaceState> emit,
  ) async {
    emit(WorkspaceLoading());
    try {
      await _workspaceService.deleteWorkspace(event.id);
      final workspaces = await _workspaceService.getWorkspaces();
      emit(WorkspaceLoaded(workspaces));
      emit(WorkspaceOperationSuccess('Workspace deleted'));
      emit(WorkspaceLoaded(workspaces));
    } catch (e) {
      emit(WorkspaceError('Failed to delete workspace: $e'));
    }
  }

  Future<void> _onAddMember(
    WorkspaceAddMember event,
    Emitter<WorkspaceState> emit,
  ) async {
    try {
      await _workspaceService.addMember(
        event.workspaceId,
        event.userId,
        role: event.role,
      );
      emit(WorkspaceOperationSuccess('Member added'));
      // Refresh list to reflect membership change.
      add(WorkspaceLoadRequested());
    } catch (e) {
      emit(WorkspaceError('Failed to add member: $e'));
    }
  }
}
