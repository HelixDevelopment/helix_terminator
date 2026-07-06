import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:shared_preferences/shared_preferences.dart';

// Events
abstract class AppEvent {}

class AppStarted extends AppEvent {}

class AppThemeChanged extends AppEvent {
  final ThemeMode themeMode;

  AppThemeChanged({required this.themeMode});
}

class AppLocaleChanged extends AppEvent {
  final Locale locale;

  AppLocaleChanged({required this.locale});
}

// States
class AppState {
  final ThemeMode themeMode;
  final Locale locale;
  final bool isInitialized;

  const AppState({
    this.themeMode = ThemeMode.system,
    this.locale = const Locale('en'),
    this.isInitialized = false,
  });

  AppState copyWith({
    ThemeMode? themeMode,
    Locale? locale,
    bool? isInitialized,
  }) {
    return AppState(
      themeMode: themeMode ?? this.themeMode,
      locale: locale ?? this.locale,
      isInitialized: isInitialized ?? this.isInitialized,
    );
  }
}

class AppBloc extends Bloc<AppEvent, AppState> {
  AppBloc() : super(const AppState()) {
    on<AppStarted>(_onAppStarted);
    on<AppThemeChanged>(_onThemeChanged);
    on<AppLocaleChanged>(_onLocaleChanged);
  }

  Future<void> _onAppStarted(
    AppStarted event,
    Emitter<AppState> emit,
  ) async {
    final prefs = await SharedPreferences.getInstance();
    final themeModeString = prefs.getString('theme_mode') ?? 'system';
    final localeString = prefs.getString('locale') ?? 'en';

    final themeMode = _parseThemeMode(themeModeString);
    final locale = Locale(localeString);

    emit(state.copyWith(
      themeMode: themeMode,
      locale: locale,
      isInitialized: true,
    ));
  }

  Future<void> _onThemeChanged(
    AppThemeChanged event,
    Emitter<AppState> emit,
  ) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('theme_mode', event.themeMode.name);
    emit(state.copyWith(themeMode: event.themeMode));
  }

  Future<void> _onLocaleChanged(
    AppLocaleChanged event,
    Emitter<AppState> emit,
  ) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('locale', event.locale.languageCode);
    emit(state.copyWith(locale: event.locale));
  }

  ThemeMode _parseThemeMode(String value) {
    switch (value) {
      case 'light':
        return ThemeMode.light;
      case 'dark':
        return ThemeMode.dark;
      default:
        return ThemeMode.system;
    }
  }
}
