import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:shared_preferences/shared_preferences.dart';

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
  static const _darkModeKey = 'settings_dark_mode';
  static const _fontSizeKey = 'settings_font_size';
  static const _pushNotificationsKey = 'settings_push_notifications';
  static const _emailNotificationsKey = 'settings_email_notifications';
  static const _biometricLockKey = 'settings_biometric_lock';
  static const _autoLockTimeoutKey = 'settings_auto_lock_timeout';
  static const _languageKey = 'settings_language';

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
      final prefs = await SharedPreferences.getInstance();
      emit(SettingsLoaded(
        darkMode: prefs.getBool(_darkModeKey) ?? false,
        fontSize: prefs.getDouble(_fontSizeKey) ?? 1.0,
        pushNotifications: prefs.getBool(_pushNotificationsKey) ?? true,
        emailNotifications: prefs.getBool(_emailNotificationsKey) ?? true,
        biometricLock: prefs.getBool(_biometricLockKey) ?? false,
        autoLockTimeout: prefs.getInt(_autoLockTimeoutKey) ?? 15,
        language: prefs.getString(_languageKey) ?? 'English',
      ));
    } catch (e) {
      emit(SettingsError(e.toString()));
    }
  }

  Future<void> _onDarkModeChanged(SettingsDarkModeChanged event, Emitter<SettingsState> emit) async {
    if (state is SettingsLoaded) {
      final current = state as SettingsLoaded;
      final updated = current.copyWith(darkMode: event.value);
      emit(updated);
      emit(SettingsActionSuccess('Dark mode ${event.value ? 'enabled' : 'disabled'}'));
      await _persistSetting(_darkModeKey, event.value);
    }
  }

  Future<void> _onFontSizeChanged(SettingsFontSizeChanged event, Emitter<SettingsState> emit) async {
    if (state is SettingsLoaded) {
      final current = state as SettingsLoaded;
      emit(current.copyWith(fontSize: event.value));
      await _persistSetting(_fontSizeKey, event.value);
    }
  }

  Future<void> _onPushNotificationsChanged(SettingsPushNotificationsChanged event, Emitter<SettingsState> emit) async {
    if (state is SettingsLoaded) {
      final current = state as SettingsLoaded;
      final updated = current.copyWith(pushNotifications: event.value);
      emit(updated);
      emit(SettingsActionSuccess('Push notifications ${event.value ? 'enabled' : 'disabled'}'));
      await _persistSetting(_pushNotificationsKey, event.value);
    }
  }

  Future<void> _onEmailNotificationsChanged(SettingsEmailNotificationsChanged event, Emitter<SettingsState> emit) async {
    if (state is SettingsLoaded) {
      final current = state as SettingsLoaded;
      final updated = current.copyWith(emailNotifications: event.value);
      emit(updated);
      emit(SettingsActionSuccess('Email notifications ${event.value ? 'enabled' : 'disabled'}'));
      await _persistSetting(_emailNotificationsKey, event.value);
    }
  }

  Future<void> _onBiometricLockChanged(SettingsBiometricLockChanged event, Emitter<SettingsState> emit) async {
    if (state is SettingsLoaded) {
      final current = state as SettingsLoaded;
      final updated = current.copyWith(biometricLock: event.value);
      emit(updated);
      emit(SettingsActionSuccess('Biometric lock ${event.value ? 'enabled' : 'disabled'}'));
      await _persistSetting(_biometricLockKey, event.value);
    }
  }

  Future<void> _onAutoLockTimeoutChanged(SettingsAutoLockTimeoutChanged event, Emitter<SettingsState> emit) async {
    if (state is SettingsLoaded) {
      final current = state as SettingsLoaded;
      final updated = current.copyWith(autoLockTimeout: event.value);
      emit(updated);
      emit(SettingsActionSuccess('Auto-lock timeout updated'));
      await _persistSetting(_autoLockTimeoutKey, event.value);
    }
  }

  Future<void> _onLanguageChanged(SettingsLanguageChanged event, Emitter<SettingsState> emit) async {
    if (state is SettingsLoaded) {
      final current = state as SettingsLoaded;
      final updated = current.copyWith(language: event.value);
      emit(updated);
      emit(SettingsActionSuccess('Language changed to ${event.value}'));
      await _persistSetting(_languageKey, event.value);
    }
  }

  Future<void> _persistSetting(String key, dynamic value) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      if (value is bool) {
        await prefs.setBool(key, value);
      } else if (value is double) {
        await prefs.setDouble(key, value);
      } else if (value is int) {
        await prefs.setInt(key, value);
      } else if (value is String) {
        await prefs.setString(key, value);
      }
    } catch (e) {
      // Silently fail persistence to avoid disrupting UI
    }
  }
}
