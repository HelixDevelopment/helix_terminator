import 'package:flutter_bloc/flutter_bloc.dart';

// TODO: define app-level events and states

class AppBloc extends Bloc<AppEvent, AppState> {
  AppBloc() : super(AppInitial()) {
    on<AppStarted>((event, emit) {
      // TODO: initialize app
      emit(AppReady());
    });
  }
}

abstract class AppEvent {}

class AppStarted extends AppEvent {}

abstract class AppState {}

class AppInitial extends AppState {}

class AppReady extends AppState {}
