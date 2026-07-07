import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'design_system/design_system.dart';
import 'bloc/ai_bloc.dart';
import 'bloc/app_bloc.dart';
import 'bloc/audit_bloc.dart';
import 'bloc/auth_bloc.dart';
import 'bloc/billing_bloc.dart';
import 'bloc/collaboration_bloc.dart';
import 'bloc/container_bridge_bloc.dart';
import 'bloc/host_bloc.dart';
import 'bloc/notification_bloc.dart';
import 'bloc/org_bloc.dart';
import 'bloc/port_forward_bloc.dart';
import 'bloc/recording_bloc.dart';
import 'bloc/settings_bloc.dart';
import 'bloc/sftp_bloc.dart';
import 'bloc/snippet_bloc.dart';
import 'bloc/terminal_bloc.dart';
import 'bloc/user_bloc.dart';
import 'bloc/vault_bloc.dart';
import 'bloc/workspace_bloc.dart';
import 'services/ai_service.dart';
import 'services/api_client.dart';
import 'services/audit_service.dart';
import 'services/auth_service.dart';
import 'services/billing_service.dart';
import 'services/collaboration_service.dart';
import 'services/host_service.dart';
import 'services/notification_service.dart';
import 'services/org_service.dart';
import 'services/port_forward_service.dart';
import 'services/recording_service.dart';
import 'services/sftp_service.dart';
import 'services/snippet_service.dart';
import 'services/user_service.dart';
import 'services/vault_service.dart';
import 'services/workspace_service.dart';
import 'screens/splash_screen.dart';

void main() {
  // TODO: initialize dependency injection, logging, analytics
  runApp(const HelixTerminatorApp());
}

class HelixTerminatorApp extends StatelessWidget {
  const HelixTerminatorApp({super.key});

  @override
  Widget build(BuildContext context) {
    final apiClient = ApiClient(baseUrl: 'https://api.helix-terminator.example.com');
    final authService = AuthService(apiClient: apiClient);

    // Re-create ApiClient with auth service for token injection
    final authenticatedApiClient = ApiClient(
      baseUrl: 'https://api.helix-terminator.example.com',
      authService: authService,
    );

    return MultiRepositoryProvider(
      providers: [
        RepositoryProvider.value(value: authenticatedApiClient),
        RepositoryProvider.value(value: authService),
        RepositoryProvider(create: (_) => NotificationService(apiClient: authenticatedApiClient)),
        RepositoryProvider(create: (_) => CollaborationService(apiClient: authenticatedApiClient)),
        RepositoryProvider(create: (_) => SftpService(apiClient: authenticatedApiClient)),
        RepositoryProvider(create: (_) => PortForwardService(apiClient: authenticatedApiClient)),
        RepositoryProvider(create: (_) => RecordingService(apiClient: authenticatedApiClient)),
        RepositoryProvider(create: (_) => SnippetService(apiClient: authenticatedApiClient)),
        RepositoryProvider(create: (_) => AiService(apiClient: authenticatedApiClient)),
        RepositoryProvider(create: (_) => AuditService(apiClient: authenticatedApiClient)),
        RepositoryProvider(create: (_) => BillingService(apiClient: authenticatedApiClient)),
        RepositoryProvider(create: (_) => OrgService(apiClient: authenticatedApiClient)),
        RepositoryProvider(create: (_) => UserService(apiClient: authenticatedApiClient)),
        RepositoryProvider(create: (_) => HostService(apiClient: authenticatedApiClient)),
        RepositoryProvider(create: (_) => VaultService(apiClient: authenticatedApiClient)),
        RepositoryProvider(create: (_) => WorkspaceService(apiClient: authenticatedApiClient)),
      ],
      child: MultiBlocProvider(
        providers: [
          BlocProvider(create: (_) => AppBloc()),
          BlocProvider(create: (context) => AuthBloc(authService: context.read<AuthService>())),
          BlocProvider(create: (context) => NotificationBloc(service: context.read<NotificationService>())),
          BlocProvider(create: (context) => CollaborationBloc(service: context.read<CollaborationService>())),
          BlocProvider(create: (context) => SftpBloc(service: context.read<SftpService>())),
          BlocProvider(create: (context) => PortForwardBloc(service: context.read<PortForwardService>())),
          BlocProvider(create: (context) => RecordingBloc(service: context.read<RecordingService>())),
          BlocProvider(create: (context) => SnippetBloc(service: context.read<SnippetService>())),
          BlocProvider(create: (context) => AiBloc(service: context.read<AiService>())),
          BlocProvider(create: (context) => AuditBloc(service: context.read<AuditService>())),
          BlocProvider(create: (context) => BillingBloc(service: context.read<BillingService>())),
          BlocProvider(create: (context) => OrgBloc(service: context.read<OrgService>())),
          BlocProvider(create: (context) => UserBloc(service: context.read<UserService>())),
          BlocProvider(create: (_) => SettingsBloc()),
          BlocProvider(create: (context) => HostBloc(service: context.read<HostService>())),
          BlocProvider(create: (context) => VaultBloc(service: context.read<VaultService>())),
          BlocProvider(create: (context) => WorkspaceBloc(service: context.read<WorkspaceService>())),
          BlocProvider(create: (context) => TerminalBloc()),
          BlocProvider(create: (context) => ContainerBridgeBloc(apiClient: context.read<ApiClient>())),
        ],
        child: MaterialApp(
          title: 'HelixTerminator',
          debugShowCheckedModeBanner: false,
          theme: HTTheme.light(),
          darkTheme: HTTheme.dark(),
          themeMode: ThemeMode.system,
          home: const SplashScreen(),
        ),
      ),
    );
  }
}
