import 'package:flutter/material.dart';
import '../bloc/auth_bloc.dart';
import 'login_screen.dart';

class SsoScreen extends StatelessWidget {
  const SsoScreen({super.key});

  void _showComingSoon(BuildContext context, String provider) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text('$provider SSO integration coming soon'),
        behavior: SnackBarBehavior.floating,
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    final providers = [
      _SsoProvider(
        name: 'Google',
        icon: Icons.g_mobiledata,
        color: const Color(0xFFEA4335),
      ),
      _SsoProvider(
        name: 'Microsoft',
        icon: Icons.window,
        color: const Color(0xFF0078D4),
      ),
      _SsoProvider(
        name: 'GitHub',
        icon: Icons.code,
        color: const Color(0xFF24292E),
      ),
      _SsoProvider(
        name: 'Okta',
        icon: Icons.verified_user,
        color: const Color(0xFF007DC1),
      ),
      _SsoProvider(
        name: 'SAML',
        icon: Icons.security,
        color: colorScheme.primary,
      ),
    ];

    return Scaffold(
      appBar: AppBar(
        title: const Text('Single Sign-On'),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => Navigator.pop(context),
        ),
      ),
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(24.0),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 400),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Icon(
                    Icons.account_circle_outlined,
                    size: 64,
                    color: colorScheme.primary,
                  ),
                  const SizedBox(height: 24),
                  Text(
                    'Sign in with SSO',
                    textAlign: TextAlign.center,
                    style: theme.textTheme.headlineSmall?.copyWith(
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                  const SizedBox(height: 8),
                  Text(
                    'Choose your organization\'s identity provider',
                    textAlign: TextAlign.center,
                    style: theme.textTheme.bodyMedium?.copyWith(
                      color: colorScheme.onSurfaceVariant,
                    ),
                  ),
                  const SizedBox(height: 32),
                  ...providers.map((provider) {
                    return Padding(
                      padding: const EdgeInsets.only(bottom: 12.0),
                      child: OutlinedButton.icon(
                        onPressed: () => _showComingSoon(context, provider.name),
                        icon: Icon(provider.icon, color: provider.color),
                        label: Text('Continue with ${provider.name}'),
                        style: OutlinedButton.styleFrom(
                          padding: const EdgeInsets.symmetric(vertical: 16),
                          alignment: Alignment.centerLeft,
                        ),
                      ),
                    );
                  }),
                  const SizedBox(height: 24),
                  Row(
                    children: [
                      Expanded(child: Divider(color: colorScheme.outlineVariant)),
                      Padding(
                        padding: const EdgeInsets.symmetric(horizontal: 16.0),
                        child: Text(
                          'OR',
                          style: TextStyle(color: colorScheme.onSurfaceVariant),
                        ),
                      ),
                      Expanded(child: Divider(color: colorScheme.outlineVariant)),
                    ],
                  ),
                  const SizedBox(height: 24),
                  TextButton(
                    onPressed: () {
                      Navigator.pushReplacement(
                        context,
                        MaterialPageRoute(
                          builder: (_) => const LoginScreen(),
                        ),
                      );
                    },
                    child: const Text('Sign in with email and password'),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _SsoProvider {
  final String name;
  final IconData icon;
  final Color color;

  _SsoProvider({
    required this.name,
    required this.icon,
    required this.color,
  });
}
