import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'bloc/app_bloc.dart';
import 'bloc/auth_bloc.dart';
import 'services/api_client.dart';
import 'services/auth_service.dart';
import 'screens/splash_screen.dart';
import 'themes/dark_theme.dart';
import 'themes/light_theme.dart';

void main() {
  final apiClient = ApiClient(baseUrl: 'https://api.helixterminator.local');
  final authService = AuthService(apiClient: apiClient);

  runApp(HelixTerminatorApp(authService: authService));
}

class HelixTerminatorApp extends StatelessWidget {
  final AuthService authService;

  const HelixTerminatorApp({super.key, required this.authService});

  @override
  Widget build(BuildContext context) {
    return MultiBlocProvider(
      providers: [
        BlocProvider(create: (_) => AppBloc()..add(AppStarted())),
        BlocProvider(create: (_) => AuthBloc(authService: authService)),
      ],
      child: BlocBuilder<AppBloc, AppState>(
        builder: (context, appState) {
          return MaterialApp(
            title: 'HelixTerminator',
            debugShowCheckedModeBanner: false,
            theme: lightTheme,
            darkTheme: darkTheme,
            themeMode: appState.themeMode,
            locale: appState.locale,
            home: const SplashScreen(),
          );
        },
      ),
    );
  }
}
