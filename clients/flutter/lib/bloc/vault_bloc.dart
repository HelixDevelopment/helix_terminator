import 'package:flutter_bloc/flutter_bloc.dart';

// TODO: define vault events and states

class VaultBloc extends Bloc<VaultEvent, VaultState> {
  VaultBloc() : super(VaultInitial()) {
    on<VaultSecretsRequested>((event, emit) async {
      // TODO: fetch secrets from API
      emit(VaultLoaded([]));
    });
  }
}

abstract class VaultEvent {}

class VaultSecretsRequested extends VaultEvent {}

abstract class VaultState {}

class VaultInitial extends VaultState {}
class VaultLoaded extends VaultState {
  final List<dynamic> secrets;
  VaultLoaded(this.secrets);
}
