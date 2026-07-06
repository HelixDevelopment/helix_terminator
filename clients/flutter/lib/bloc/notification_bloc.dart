import 'package:flutter_bloc/flutter_bloc.dart';
import '../models/notification.dart';
import '../services/notification_service.dart';

// Events
abstract class NotificationEvent {}

class NotificationListRequested extends NotificationEvent {}

class NotificationFilterChanged extends NotificationEvent {
  final String? type;
  final bool? unreadOnly;
  NotificationFilterChanged({this.type, this.unreadOnly});
}

class NotificationMarkAsRead extends NotificationEvent {
  final String id;
  NotificationMarkAsRead(this.id);
}

class NotificationMarkAllAsRead extends NotificationEvent {}

class NotificationDelete extends NotificationEvent {
  final String id;
  NotificationDelete(this.id);
}

class NotificationSearchChanged extends NotificationEvent {
  final String query;
  NotificationSearchChanged(this.query);
}

// States
abstract class NotificationState {}

class NotificationInitial extends NotificationState {}

class NotificationLoading extends NotificationState {}

class NotificationLoaded extends NotificationState {
  final List<Notification> notifications;
  final String? filterType;
  final bool? unreadOnly;
  final String searchQuery;
  NotificationLoaded(this.notifications, {this.filterType, this.unreadOnly, this.searchQuery = ''});
}

class NotificationError extends NotificationState {
  final String message;
  NotificationError(this.message);
}

class NotificationActionSuccess extends NotificationState {
  final String message;
  NotificationActionSuccess(this.message);
}

// Bloc
class NotificationBloc extends Bloc<NotificationEvent, NotificationState> {
  final NotificationService _service;

  NotificationBloc({required NotificationService service})
      : _service = service,
        super(NotificationInitial()) {
    on<NotificationListRequested>(_onListRequested);
    on<NotificationFilterChanged>(_onFilterChanged);
    on<NotificationMarkAsRead>(_onMarkAsRead);
    on<NotificationMarkAllAsRead>(_onMarkAllAsRead);
    on<NotificationDelete>(_onDelete);
    on<NotificationSearchChanged>(_onSearchChanged);
  }

  Future<void> _onListRequested(
    NotificationListRequested event,
    Emitter<NotificationState> emit,
  ) async {
    emit(NotificationLoading());
    try {
      final notifications = await _service.getNotifications();
      emit(NotificationLoaded(notifications));
    } catch (e) {
      emit(NotificationError(e.toString()));
    }
  }

  Future<void> _onFilterChanged(
    NotificationFilterChanged event,
    Emitter<NotificationState> emit,
  ) async {
    emit(NotificationLoading());
    try {
      final notifications = await _service.getNotifications(
        type: event.type,
        unreadOnly: event.unreadOnly,
      );
      emit(NotificationLoaded(
        notifications,
        filterType: event.type,
        unreadOnly: event.unreadOnly,
      ));
    } catch (e) {
      emit(NotificationError(e.toString()));
    }
  }

  Future<void> _onMarkAsRead(
    NotificationMarkAsRead event,
    Emitter<NotificationState> emit,
  ) async {
    try {
      await _service.markAsRead(event.id);
      emit(NotificationActionSuccess('Marked as read'));
      add(NotificationListRequested());
    } catch (e) {
      emit(NotificationError(e.toString()));
    }
  }

  Future<void> _onMarkAllAsRead(
    NotificationMarkAllAsRead event,
    Emitter<NotificationState> emit,
  ) async {
    try {
      await _service.markAllAsRead();
      emit(NotificationActionSuccess('All marked as read'));
      add(NotificationListRequested());
    } catch (e) {
      emit(NotificationError(e.toString()));
    }
  }

  Future<void> _onDelete(
    NotificationDelete event,
    Emitter<NotificationState> emit,
  ) async {
    try {
      await _service.deleteNotification(event.id);
      emit(NotificationActionSuccess('Notification deleted'));
      add(NotificationListRequested());
    } catch (e) {
      emit(NotificationError(e.toString()));
    }
  }

  Future<void> _onSearchChanged(
    NotificationSearchChanged event,
    Emitter<NotificationState> emit,
  ) async {
    // Search is handled client-side for notifications
    if (state is NotificationLoaded) {
      final current = state as NotificationLoaded;
      emit(NotificationLoaded(
        current.notifications,
        filterType: current.filterType,
        unreadOnly: current.unreadOnly,
        searchQuery: event.query,
      ));
    }
  }
}
