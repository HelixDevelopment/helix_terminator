import 'package:flutter_bloc/flutter_bloc.dart';

// TODO: define host events and states

class HostBloc extends Bloc<HostEvent, HostState> {
  HostBloc() : super(HostInitial()) {
    on<HostListRequested>((event, emit) async {
      // TODO: fetch host list from API
      emit(HostLoaded([]));
    });
  }
}

abstract class HostEvent {}

class HostListRequested extends HostEvent {}

abstract class HostState {}

class HostInitial extends HostState {}
class HostLoaded extends HostState {
  final List<dynamic> hosts;
  HostLoaded(this.hosts);
}
