import 'package:flutter_bloc/flutter_bloc.dart';
import '../models/audit_log.dart';
import '../services/audit_service.dart';

// Events
abstract class AuditEvent {}

class AuditLogListRequested extends AuditEvent {}

class AuditLogFilterChanged extends AuditEvent {
  final String? action;
  final DateTime? from;
  final DateTime? to;
  AuditLogFilterChanged({this.action, this.from, this.to});
}

class AuditLogSearchChanged extends AuditEvent {
  final String query;
  AuditLogSearchChanged(this.query);
}

class AuditLogExportRequested extends AuditEvent {
  final String? action;
  final DateTime? from;
  final DateTime? to;
  AuditLogExportRequested({this.action, this.from, this.to});
}

// States
abstract class AuditState {}

class AuditInitial extends AuditState {}

class AuditLoading extends AuditState {}

class AuditListLoaded extends AuditState {
  final List<AuditLog> logs;
  final String searchQuery;
  AuditListLoaded(this.logs, {this.searchQuery = ''});
}

class AuditError extends AuditState {
  final String message;
  AuditError(this.message);
}

class AuditActionSuccess extends AuditState {
  final String message;
  AuditActionSuccess(this.message);
}

// Bloc
class AuditBloc extends Bloc<AuditEvent, AuditState> {
  final AuditService _service;

  AuditBloc({required AuditService service})
      : _service = service,
        super(AuditInitial()) {
    on<AuditLogListRequested>(_onListRequested);
    on<AuditLogFilterChanged>(_onFilterChanged);
    on<AuditLogSearchChanged>(_onSearchChanged);
    on<AuditLogExportRequested>(_onExportRequested);
  }

  Future<void> _onListRequested(AuditLogListRequested event, Emitter<AuditState> emit) async {
    emit(AuditLoading());
    try {
      final logs = await _service.getAuditLogs();
      emit(AuditListLoaded(logs));
    } catch (e) {
      emit(AuditError(e.toString()));
    }
  }

  Future<void> _onFilterChanged(AuditLogFilterChanged event, Emitter<AuditState> emit) async {
    emit(AuditLoading());
    try {
      final logs = await _service.getAuditLogs(
        action: event.action,
        from: event.from,
        to: event.to,
      );
      emit(AuditListLoaded(logs));
    } catch (e) {
      emit(AuditError(e.toString()));
    }
  }

  Future<void> _onSearchChanged(AuditLogSearchChanged event, Emitter<AuditState> emit) async {
    if (state is AuditListLoaded) {
      final current = state as AuditListLoaded;
      emit(AuditListLoaded(current.logs, searchQuery: event.query));
    }
  }

  Future<void> _onExportRequested(AuditLogExportRequested event, Emitter<AuditState> emit) async {
    try {
      final url = await _service.exportAuditLogs(
        action: event.action,
        from: event.from,
        to: event.to,
      );
      emit(AuditActionSuccess('Export ready: $url'));
    } catch (e) {
      emit(AuditError(e.toString()));
    }
  }
}
