import 'package:flutter_bloc/flutter_bloc.dart';

// TODO: define notification events and states

class NotificationBloc extends Bloc<NotificationEvent, NotificationState> {
  NotificationBloc() : super(NotificationInitial()) {
    on<NotificationListRequested>((event, emit) async {
      // TODO: fetch notifications from API
      emit(NotificationLoaded([]));
    });
  }
}

abstract class NotificationEvent {}

class NotificationListRequested extends NotificationEvent {}

abstract class NotificationState {}

class NotificationInitial extends NotificationState {}
class NotificationLoaded extends NotificationState {
  final List<dynamic> notifications;
  NotificationLoaded(this.notifications);
}
