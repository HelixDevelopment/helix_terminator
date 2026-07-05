import 'package:flutter_bloc/flutter_bloc.dart';

// TODO: define workspace events and states

class WorkspaceBloc extends Bloc<WorkspaceEvent, WorkspaceState> {
  WorkspaceBloc() : super(WorkspaceInitial()) {
    on<WorkspaceListRequested>((event, emit) async {
      // TODO: fetch workspaces from API
      emit(WorkspaceLoaded([]));
    });
  }
}

abstract class WorkspaceEvent {}

class WorkspaceListRequested extends WorkspaceEvent {}

abstract class WorkspaceState {}

class WorkspaceInitial extends WorkspaceState {}
class WorkspaceLoaded extends WorkspaceState {
  final List<dynamic> workspaces;
  WorkspaceLoaded(this.workspaces);
}
