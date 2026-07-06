import 'package:flutter_bloc/flutter_bloc.dart';
import '../services/ai_service.dart';

// Events
abstract class AiEvent {}

class AiSendMessage extends AiEvent {
  final String message;
  final List<Map<String, String>>? history;
  AiSendMessage(this.message, {this.history});
}

class AiGetSuggestions extends AiEvent {
  final String context;
  AiGetSuggestions(this.context);
}

// States
abstract class AiState {}

class AiInitial extends AiState {}

class AiLoading extends AiState {}

class AiMessageReceived extends AiState {
  final String message;
  AiMessageReceived(this.message);
}

class AiSuggestionsLoaded extends AiState {
  final List<String> suggestions;
  AiSuggestionsLoaded(this.suggestions);
}

class AiError extends AiState {
  final String message;
  AiError(this.message);
}

// Bloc
class AiBloc extends Bloc<AiEvent, AiState> {
  final AiService _service;

  AiBloc({required AiService service})
      : _service = service,
        super(AiInitial()) {
    on<AiSendMessage>(_onSendMessage);
    on<AiGetSuggestions>(_onGetSuggestions);
  }

  Future<void> _onSendMessage(AiSendMessage event, Emitter<AiState> emit) async {
    emit(AiLoading());
    try {
      final response = await _service.sendMessage(event.message, history: event.history);
      emit(AiMessageReceived(response));
    } catch (e) {
      emit(AiError(e.toString()));
    }
  }

  Future<void> _onGetSuggestions(AiGetSuggestions event, Emitter<AiState> emit) async {
    emit(AiLoading());
    try {
      final suggestions = await _service.getSuggestions(event.context);
      emit(AiSuggestionsLoaded(suggestions));
    } catch (e) {
      emit(AiError(e.toString()));
    }
  }
}
