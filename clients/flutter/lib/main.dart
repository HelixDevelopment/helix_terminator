import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'bloc/app_bloc.dart';
import 'screens/splash_screen.dart';
import 'themes/dark_theme.dart';
import 'themes/light_theme.dart';

void main() {
  // TODO: initialize dependency injection, logging, analytics
  runApp(const HelixTerminatorApp());
}

class HelixTerminatorApp extends StatelessWidget {
  const HelixTerminatorApp({super.key});

  @override
  Widget build(BuildContext context) {
    return BlocProvider(
      create: (_) => AppBloc(),
      child: MaterialApp(
        title: 'HelixTerminator',
        debugShowCheckedModeBanner: false,
        theme: lightTheme,
        darkTheme: darkTheme,
        themeMode: ThemeMode.system,
        home: const SplashScreen(),
      ),
    );
  }
}
