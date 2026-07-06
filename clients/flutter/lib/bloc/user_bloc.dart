import 'package:flutter_bloc/flutter_bloc.dart';
import '../models/user.dart';
import '../services/user_service.dart';

// Events
abstract class UserEvent {}

class UserProfileRequested extends UserEvent {}

class UserUpdateProfile extends UserEvent {
  final String? name;
  final String? avatarUrl;
  UserUpdateProfile({this.name, this.avatarUrl});
}

class UserChangePassword extends UserEvent {
  final String currentPassword;
  final String newPassword;
  UserChangePassword(this.currentPassword, this.newPassword);
}

class UserEnableTwoFactor extends UserEvent {}

class UserDisableTwoFactor extends UserEvent {
  final String code;
  UserDisableTwoFactor(this.code);
}

class UserUpdatePreferences extends UserEvent {
  final Map<String, dynamic> preferences;
  UserUpdatePreferences(this.preferences);
}

// States
abstract class UserState {}

class UserInitial extends UserState {}

class UserLoading extends UserState {}

class UserProfileLoaded extends UserState {
  final User user;
  final bool twoFactorEnabled;
  UserProfileLoaded(this.user, {this.twoFactorEnabled = false});
}

class UserError extends UserState {
  final String message;
  UserError(this.message);
}

class UserActionSuccess extends UserState {
  final String message;
  UserActionSuccess(this.message);
}

// Bloc
class UserBloc extends Bloc<UserEvent, UserState> {
  final UserService _service;

  UserBloc({required UserService service})
      : _service = service,
        super(UserInitial()) {
    on<UserProfileRequested>(_onProfileRequested);
    on<UserUpdateProfile>(_onUpdateProfile);
    on<UserChangePassword>(_onChangePassword);
    on<UserEnableTwoFactor>(_onEnableTwoFactor);
    on<UserDisableTwoFactor>(_onDisableTwoFactor);
    on<UserUpdatePreferences>(_onUpdatePreferences);
  }

  Future<void> _onProfileRequested(UserProfileRequested event, Emitter<UserState> emit) async {
    emit(UserLoading());
    try {
      final user = await _service.getCurrentUser();
      final preferences = await _service.getPreferences();
      final twoFactorEnabled = preferences['twoFactorEnabled'] as bool? ?? false;
      emit(UserProfileLoaded(user, twoFactorEnabled: twoFactorEnabled));
    } catch (e) {
      emit(UserError(e.toString()));
    }
  }

  Future<void> _onUpdateProfile(UserUpdateProfile event, Emitter<UserState> emit) async {
    try {
      final user = await _service.updateProfile(name: event.name, avatarUrl: event.avatarUrl);
      emit(UserActionSuccess('Profile updated'));
      emit(UserProfileLoaded(user));
    } catch (e) {
      emit(UserError(e.toString()));
    }
  }

  Future<void> _onChangePassword(UserChangePassword event, Emitter<UserState> emit) async {
    try {
      await _service.changePassword(event.currentPassword, event.newPassword);
      emit(UserActionSuccess('Password changed'));
    } catch (e) {
      emit(UserError(e.toString()));
    }
  }

  Future<void> _onEnableTwoFactor(UserEnableTwoFactor event, Emitter<UserState> emit) async {
    try {
      await _service.enableTwoFactor();
      emit(UserActionSuccess('2FA enabled'));
      add(UserProfileRequested());
    } catch (e) {
      emit(UserError(e.toString()));
    }
  }

  Future<void> _onDisableTwoFactor(UserDisableTwoFactor event, Emitter<UserState> emit) async {
    try {
      await _service.disableTwoFactor(event.code);
      emit(UserActionSuccess('2FA disabled'));
      add(UserProfileRequested());
    } catch (e) {
      emit(UserError(e.toString()));
    }
  }

  Future<void> _onUpdatePreferences(UserUpdatePreferences event, Emitter<UserState> emit) async {
    try {
      await _service.updatePreferences(event.preferences);
      emit(UserActionSuccess('Preferences updated'));
      add(UserProfileRequested());
    } catch (e) {
      emit(UserError(e.toString()));
    }
  }
}
