import 'package:flutter_bloc/flutter_bloc.dart';

// TODO: define auth events and states

class AuthBloc extends Bloc<AuthEvent, AuthState> {
  AuthBloc() : super(AuthInitial()) {
    on<AuthLoginRequested>((event, emit) async {
      // TODO: call auth API
      emit(AuthAuthenticated());
    });
    on<AuthLogoutRequested>((event, emit) {
      emit(AuthUnauthenticated());
    });
  }
}

abstract class AuthEvent {}

class AuthLoginRequested extends AuthEvent {}
class AuthLogoutRequested extends AuthEvent {}

abstract class AuthState {}

class AuthInitial extends AuthState {}
class AuthAuthenticated extends AuthState {}
class AuthUnauthenticated extends AuthState {}
