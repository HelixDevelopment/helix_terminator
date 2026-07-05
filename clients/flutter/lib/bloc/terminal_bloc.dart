import 'package:flutter_bloc/flutter_bloc.dart';

// TODO: define terminal events and states

class TerminalBloc extends Bloc<TerminalEvent, TerminalState> {
  TerminalBloc() : super(TerminalInitial()) {
    on<TerminalConnected>((event, emit) {
      emit(TerminalReady());
    });
  }
}

abstract class TerminalEvent {}

class TerminalConnected extends TerminalEvent {}

abstract class TerminalState {}

class TerminalInitial extends TerminalState {}
class TerminalReady extends TerminalState {}
