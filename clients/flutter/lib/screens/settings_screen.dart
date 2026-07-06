import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import '../bloc/settings_bloc.dart';
import '../widgets/loading_indicator.dart';

class SettingsScreen extends StatefulWidget {
  const SettingsScreen({super.key});

  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends State<SettingsScreen> {
  @override
  void initState() {
    super.initState();
    context.read<SettingsBloc>().add(SettingsLoadRequested());
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Settings'),
      ),
      body: BlocConsumer<SettingsBloc, SettingsState>(
        listener: (context, state) {
          if (state is SettingsActionSuccess) {
            ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(state.message)));
          }
        },
        builder: (context, state) {
          if (state is SettingsLoading) {
            return const LoadingIndicator();
          }
          if (state is SettingsLoaded) {
            return ListView(
              children: [
                // Appearance
                _SettingsSection(
                  title: 'Appearance',
                  children: [
                    ListTile(
                      leading: const Icon(Icons.dark_mode),
                      title: const Text('Dark Mode'),
                      trailing: Switch(
                        value: state.darkMode,
                        onChanged: (value) {
                          context.read<SettingsBloc>().add(SettingsDarkModeChanged(value));
                        },
                      ),
                    ),
                    ListTile(
                      leading: const Icon(Icons.format_size),
                      title: const Text('Font Size'),
                      subtitle: Text('${state.fontSize.toStringAsFixed(1)}x'),
                      trailing: SizedBox(
                        width: 150,
                        child: Slider(
                          value: state.fontSize,
                          min: 0.8,
                          max: 1.5,
                          divisions: 7,
                          onChanged: (value) {
                            context.read<SettingsBloc>().add(SettingsFontSizeChanged(value));
                          },
                        ),
                      ),
                    ),
                  ],
                ),

                // Notifications
                _SettingsSection(
                  title: 'Notifications',
                  children: [
                    ListTile(
                      leading: const Icon(Icons.notifications),
                      title: const Text('Push Notifications'),
                      trailing: Switch(
                        value: state.pushNotifications,
                        onChanged: (value) {
                          context.read<SettingsBloc>().add(SettingsPushNotificationsChanged(value));
                        },
                      ),
                    ),
                    ListTile(
                      leading: const Icon(Icons.email),
                      title: const Text('Email Notifications'),
                      trailing: Switch(
                        value: state.emailNotifications,
                        onChanged: (value) {
                          context.read<SettingsBloc>().add(SettingsEmailNotificationsChanged(value));
                        },
                      ),
                    ),
                  ],
                ),

                // Security
                _SettingsSection(
                  title: 'Security',
                  children: [
                    ListTile(
                      leading: const Icon(Icons.fingerprint),
                      title: const Text('Biometric Lock'),
                      trailing: Switch(
                        value: state.biometricLock,
                        onChanged: (value) {
                          context.read<SettingsBloc>().add(SettingsBiometricLockChanged(value));
                        },
                      ),
                    ),
                    ListTile(
                      leading: const Icon(Icons.lock_clock),
                      title: const Text('Auto-Lock Timeout'),
                      subtitle: Text('${state.autoLockTimeout} minutes'),
                      trailing: const Icon(Icons.chevron_right),
                      onTap: () => _showAutoLockDialog(state.autoLockTimeout),
                    ),
                  ],
                ),

                // Language
                _SettingsSection(
                  title: 'Language',
                  children: [
                    ListTile(
                      leading: const Icon(Icons.language),
                      title: const Text('Language'),
                      subtitle: Text(state.language),
                      trailing: const Icon(Icons.chevron_right),
                      onTap: () => _showLanguageDialog(state.language),
                    ),
                  ],
                ),

                // About
                _SettingsSection(
                  title: 'About',
                  children: [
                    ListTile(
                      leading: const Icon(Icons.info),
                      title: const Text('Version'),
                      subtitle: const Text('0.1.0+1'),
                    ),
                    ListTile(
                      leading: const Icon(Icons.description),
                      title: const Text('Terms of Service'),
                      trailing: const Icon(Icons.chevron_right),
                      onTap: () {},
                    ),
                    ListTile(
                      leading: const Icon(Icons.privacy_tip),
                      title: const Text('Privacy Policy'),
                      trailing: const Icon(Icons.chevron_right),
                      onTap: () {},
                    ),
                  ],
                ),
              ],
            );
          }
          return const SizedBox.shrink();
        },
      ),
    );
  }

  void _showAutoLockDialog(int currentValue) {
    final values = [1, 5, 10, 15, 30, 60];
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Auto-Lock Timeout'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: values.map((v) => RadioListTile<int>(
            title: Text('$v minute${v > 1 ? 's' : ''}'),
            value: v,
            groupValue: currentValue,
            onChanged: (value) {
              if (value != null) {
                context.read<SettingsBloc>().add(SettingsAutoLockTimeoutChanged(value));
                Navigator.pop(context);
              }
            },
          )).toList(),
        ),
      ),
    );
  }

  void _showLanguageDialog(String currentLanguage) {
    final languages = ['English', 'Spanish', 'French', 'German', 'Chinese', 'Japanese'];
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Select Language'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: languages.map((lang) => RadioListTile<String>(
            title: Text(lang),
            value: lang,
            groupValue: currentLanguage,
            onChanged: (value) {
              if (value != null) {
                context.read<SettingsBloc>().add(SettingsLanguageChanged(value));
                Navigator.pop(context);
              }
            },
          )).toList(),
        ),
      ),
    );
  }
}

class _SettingsSection extends StatelessWidget {
  final String title;
  final List<Widget> children;

  const _SettingsSection({required this.title, required this.children});

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 24, 16, 8),
          child: Text(
            title.toUpperCase(),
            style: Theme.of(context).textTheme.labelLarge?.copyWith(
              color: Theme.of(context).colorScheme.primary,
              fontWeight: FontWeight.bold,
            ),
          ),
        ),
        ...children,
      ],
    );
  }
}
