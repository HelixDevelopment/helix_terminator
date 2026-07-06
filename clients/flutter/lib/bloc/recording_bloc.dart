import 'package:flutter_bloc/flutter_bloc.dart';
import '../models/recording.dart';
import '../services/recording_service.dart';

// Events
abstract class RecordingEvent {}

class RecordingListRequested extends RecordingEvent {}

class RecordingLoadRequested extends RecordingEvent {
  final String id;
  RecordingLoadRequested(this.id);
}

class RecordingDelete extends RecordingEvent {
  final String id;
  RecordingDelete(this.id);
}

class RecordingSearchChanged extends RecordingEvent {
  final String query;
  RecordingSearchChanged(this.query);
}

// States
abstract class RecordingState {}

class RecordingInitial extends RecordingState {}

class RecordingLoading extends RecordingState {}

class RecordingListLoaded extends RecordingState {
  final List<Recording> recordings;
  final String searchQuery;
  RecordingListLoaded(this.recordings, {this.searchQuery = ''});
}

class RecordingDetailLoaded extends RecordingState {
  final Recording recording;
  RecordingDetailLoaded(this.recording);
}

class RecordingError extends RecordingState {
  final String message;
  RecordingError(this.message);
}

class RecordingActionSuccess extends RecordingState {
  final String message;
  RecordingActionSuccess(this.message);
}

// Bloc
class RecordingBloc extends Bloc<RecordingEvent, RecordingState> {
  final RecordingService _service;

  RecordingBloc({required RecordingService service})
      : _service = service,
        super(RecordingInitial()) {
    on<RecordingListRequested>(_onListRequested);
    on<RecordingLoadRequested>(_onLoadRequested);
    on<RecordingDelete>(_onDelete);
    on<RecordingSearchChanged>(_onSearchChanged);
  }

  Future<void> _onListRequested(RecordingListRequested event, Emitter<RecordingState> emit) async {
    emit(RecordingLoading());
    try {
      final recordings = await _service.getRecordings();
      emit(RecordingListLoaded(recordings));
    } catch (e) {
      emit(RecordingError(e.toString()));
    }
  }

  Future<void> _onLoadRequested(RecordingLoadRequested event, Emitter<RecordingState> emit) async {
    emit(RecordingLoading());
    try {
      final recording = await _service.getRecording(event.id);
      emit(RecordingDetailLoaded(recording));
    } catch (e) {
      emit(RecordingError(e.toString()));
    }
  }

  Future<void> _onDelete(RecordingDelete event, Emitter<RecordingState> emit) async {
    try {
      await _service.deleteRecording(event.id);
      emit(RecordingActionSuccess('Recording deleted'));
      add(RecordingListRequested());
    } catch (e) {
      emit(RecordingError(e.toString()));
    }
  }

  Future<void> _onSearchChanged(RecordingSearchChanged event, Emitter<RecordingState> emit) async {
    if (state is RecordingListLoaded) {
      final current = state as RecordingListLoaded;
      emit(RecordingListLoaded(current.recordings, searchQuery: event.query));
    }
  }
}
