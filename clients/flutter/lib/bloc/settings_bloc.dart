import 'package:flutter_bloc/flutter_bloc.dart';

// Events
abstract class SettingsEvent {}

class SettingsLoadRequested extends SettingsEvent {}

class SettingsDarkModeChanged extends SettingsEvent {
  final bool value;
  SettingsDarkModeChanged(this.value);
}

class SettingsFontSizeChanged extends SettingsEvent {
  final double value;
  SettingsFontSizeChanged(this.value);
}

class SettingsPushNotificationsChanged extends SettingsEvent {
  final bool value;
  SettingsPushNotificationsChanged(this.value);
}

class SettingsEmailNotificationsChanged extends SettingsEvent {
  final bool value;
  SettingsEmailNotificationsChanged(this.value);
}

class SettingsBiometricLockChanged extends SettingsEvent {
  final bool value;
  SettingsBiometricLockChanged(this.value);
}

class SettingsAutoLockTimeoutChanged extends SettingsEvent {
  final int value;
  SettingsAutoLockTimeoutChanged(this.value);
}

class SettingsLanguageChanged extends SettingsEvent {
  final String value;
  SettingsLanguageChanged(this.value);
}

// States
abstract class SettingsState {}

class SettingsInitial extends SettingsState {}

class SettingsLoading extends SettingsState {}

class SettingsLoaded extends SettingsState {
  final bool darkMode;
  final double fontSize;
  final bool pushNotifications;
  final bool emailNotifications;
  final bool biometricLock;
  final int autoLockTimeout;
  final String language;

  SettingsLoaded({
    this.darkMode = false,
    this.fontSize = 1.0,
    this.pushNotifications = true,
    this.emailNotifications = true,
    this.biometricLock = false,
    this.autoLockTimeout = 15,
    this.language = 'English',
  });

  SettingsLoaded copyWith({
    bool? darkMode,
    double? fontSize,
    bool? pushNotifications,
    bool? emailNotifications,
    bool? biometricLock,
    int? autoLockTimeout,
    String? language,
  }) {
    return SettingsLoaded(
      darkMode: darkMode ?? this.darkMode,
      fontSize: fontSize ?? this.fontSize,
      pushNotifications: pushNotifications ?? this.pushNotifications,
      emailNotifications: emailNotifications ?? this.emailNotifications,
      biometricLock: biometricLock ?? this.biometricLock,
      autoLockTimeout: autoLockTimeout ?? this.autoLockTimeout,
      language: language ?? this.language,
    );
  }
}

class SettingsError extends SettingsState {
  final String message;
  SettingsError(this.message);
}

class SettingsActionSuccess extends SettingsState {
  final String message;
  SettingsActionSuccess(this.message);
}

// Bloc
class SettingsBloc extends Bloc<SettingsEvent, SettingsState> {
  SettingsBloc() : super(SettingsInitial()) {
    on<SettingsLoadRequested>(_onLoadRequested);
    on<SettingsDarkModeChanged>(_onDarkModeChanged);
    on<SettingsFontSizeChanged>(_onFontSizeChanged);
    on<SettingsPushNotificationsChanged>(_onPushNotificationsChanged);
    on<SettingsEmailNotificationsChanged>(_onEmailNotificationsChanged);
    on<SettingsBiometricLockChanged>(_onBiometricLockChanged);
    on<SettingsAutoLockTimeoutChanged>(_onAutoLockTimeoutChanged);
    on<SettingsLanguageChanged>(_onLanguageChanged);
  }

  Future<void> _onLoadRequested(SettingsLoadRequested event, Emitter<SettingsState> emit) async {
    emit(SettingsLoading());
    try {
      // TODO: load from shared preferences
      emit(SettingsLoaded());
    } catch (e) {
      emit(SettingsError(e.toString()));
    }
  }

  Future<void> _onDarkModeChanged(SettingsDarkModeChanged event, Emitter<SettingsState> emit) async {
    if (state is SettingsLoaded) {
      final current = state as SettingsLoaded;
      emit(current.copyWith(darkMode: event.value));
      emit(SettingsActionSuccess('Dark mode ${event.value ? 'enabled' : 'disabled'}'));
    }
  }

  Future<void> _onFontSizeChanged(SettingsFontSizeChanged event, Emitter<SettingsState> emit) async {
    if (state is SettingsLoaded) {
      final current = state as SettingsLoaded;
      emit(current.copyWith(fontSize: event.value));
    }
  }

  Future<void> _onPushNotificationsChanged(SettingsPushNotificationsChanged event, Emitter<SettingsState> emit) async {
    if (state is SettingsLoaded) {
      final current = state as SettingsLoaded;
      emit(current.copyWith(pushNotifications: event.value));
      emit(SettingsActionSuccess('Push notifications ${event.value ? 'enabled' : 'disabled'}'));
    }
  }

  Future<void> _onEmailNotificationsChanged(SettingsEmailNotificationsChanged event, Emitter<SettingsState> emit) async {
    if (state is SettingsLoaded) {
      final current = state as SettingsLoaded;
      emit(current.copyWith(emailNotifications: event.value));
      emit(SettingsActionSuccess('Email notifications ${event.value ? 'enabled' : 'disabled'}'));
    }
  }

  Future<void> _onBiometricLockChanged(SettingsBiometricLockChanged event, Emitter<SettingsState> emit) async {
    if (state is SettingsLoaded) {
      final current = state as SettingsLoaded;
      emit(current.copyWith(biometricLock: event.value));
      emit(SettingsActionSuccess('Biometric lock ${event.value ? 'enabled' : 'disabled'}'));
    }
  }

  Future<void> _onAutoLockTimeoutChanged(SettingsAutoLockTimeoutChanged event, Emitter<SettingsState> emit) async {
    if (state is SettingsLoaded) {
      final current = state as SettingsLoaded;
      emit(current.copyWith(autoLockTimeout: event.value));
      emit(SettingsActionSuccess('Auto-lock timeout updated'));
    }
  }

  Future<void> _onLanguageChanged(SettingsLanguageChanged event, Emitter<SettingsState> emit) async {
    if (state is SettingsLoaded) {
      final current = state as SettingsLoaded;
      emit(current.copyWith(language: event.value));
      emit(SettingsActionSuccess('Language changed to ${event.value}'));
    }
  }
}
