import 'package:flutter_bloc/flutter_bloc.dart';

import '../models/secret.dart';
import '../services/vault_service.dart';

// ------------------------------------------------------------------
// Events
// ------------------------------------------------------------------

abstract class VaultEvent {}

class VaultLoadRequested extends VaultEvent {}

class VaultCreateRequested extends VaultEvent {
  final String name;
  final String value;
  final String type;
  final String? category;
  final String? description;
  VaultCreateRequested({
    required this.name,
    required this.value,
    required this.type,
    this.category,
    this.description,
  });
}

class VaultUpdateRequested extends VaultEvent {
  final String id;
  final String? name;
  final String? value;
  final String? type;
  final String? category;
  final String? description;
  VaultUpdateRequested({
    required this.id,
    this.name,
    this.value,
    this.type,
    this.category,
    this.description,
  });
}

class VaultDeleteRequested extends VaultEvent {
  final String id;
  VaultDeleteRequested(this.id);
}

// ------------------------------------------------------------------
// States
// ------------------------------------------------------------------

abstract class VaultState {}

class VaultInitial extends VaultState {}

class VaultLoading extends VaultState {}

class VaultLoaded extends VaultState {
  final List<Secret> secrets;
  VaultLoaded(this.secrets);
}

class VaultOperationSuccess extends VaultState {
  final String message;
  VaultOperationSuccess(this.message);
}

class VaultError extends VaultState {
  final String message;
  VaultError(this.message);
}

// ------------------------------------------------------------------
// BLoC
// ------------------------------------------------------------------

class VaultBloc extends Bloc<VaultEvent, VaultState> {
  final VaultService _vaultService;

  VaultBloc({VaultService? vaultService})
      : _vaultService = vaultService ?? VaultService(),
        super(VaultInitial()) {
    on<VaultLoadRequested>(_onLoad);
    on<VaultCreateRequested>(_onCreate);
    on<VaultUpdateRequested>(_onUpdate);
    on<VaultDeleteRequested>(_onDelete);
  }

  Future<void> _onLoad(
    VaultLoadRequested event,
    Emitter<VaultState> emit,
  ) async {
    emit(VaultLoading());
    try {
      final secrets = await _vaultService.getSecrets();
      emit(VaultLoaded(secrets));
    } catch (e) {
      emit(VaultError('Failed to load secrets: $e'));
    }
  }

  Future<void> _onCreate(
    VaultCreateRequested event,
    Emitter<VaultState> emit,
  ) async {
    emit(VaultLoading());
    try {
      await _vaultService.createSecret(
        name: event.name,
        value: event.value,
        type: event.type,
        category: event.category,
        description: event.description,
      );
      final secrets = await _vaultService.getSecrets();
      emit(VaultLoaded(secrets));
      emit(VaultOperationSuccess('Secret created'));
      emit(VaultLoaded(secrets));
    } catch (e) {
      emit(VaultError('Failed to create secret: $e'));
    }
  }

  Future<void> _onUpdate(
    VaultUpdateRequested event,
    Emitter<VaultState> emit,
  ) async {
    emit(VaultLoading());
    try {
      await _vaultService.updateSecret(
        event.id,
        name: event.name,
        value: event.value,
        type: event.type,
        category: event.category,
        description: event.description,
      );
      final secrets = await _vaultService.getSecrets();
      emit(VaultLoaded(secrets));
      emit(VaultOperationSuccess('Secret updated'));
      emit(VaultLoaded(secrets));
    } catch (e) {
      emit(VaultError('Failed to update secret: $e'));
    }
  }

  Future<void> _onDelete(
    VaultDeleteRequested event,
    Emitter<VaultState> emit,
  ) async {
    emit(VaultLoading());
    try {
      await _vaultService.deleteSecret(event.id);
      final secrets = await _vaultService.getSecrets();
      emit(VaultLoaded(secrets));
      emit(VaultOperationSuccess('Secret deleted'));
      emit(VaultLoaded(secrets));
    } catch (e) {
      emit(VaultError('Failed to delete secret: $e'));
    }
  }
}
