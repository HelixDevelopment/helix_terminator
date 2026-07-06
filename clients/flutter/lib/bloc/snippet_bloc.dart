import 'package:flutter_bloc/flutter_bloc.dart';
import '../models/snippet.dart';
import '../services/snippet_service.dart';

// Events
abstract class SnippetEvent {}

class SnippetListRequested extends SnippetEvent {}

class SnippetLoadRequested extends SnippetEvent {
  final String id;
  SnippetLoadRequested(this.id);
}

class SnippetCreate extends SnippetEvent {
  final String title;
  final String content;
  final String language;
  SnippetCreate({required this.title, required this.content, required this.language});
}

class SnippetUpdate extends SnippetEvent {
  final String id;
  final String? title;
  final String? content;
  final String? language;
  SnippetUpdate(this.id, {this.title, this.content, this.language});
}

class SnippetDelete extends SnippetEvent {
  final String id;
  SnippetDelete(this.id);
}

class SnippetSearchChanged extends SnippetEvent {
  final String query;
  SnippetSearchChanged(this.query);
}

// States
abstract class SnippetState {}

class SnippetInitial extends SnippetState {}

class SnippetLoading extends SnippetState {}

class SnippetListLoaded extends SnippetState {
  final List<Snippet> snippets;
  final String searchQuery;
  SnippetListLoaded(this.snippets, {this.searchQuery = ''});
}

class SnippetDetailLoaded extends SnippetState {
  final Snippet snippet;
  SnippetDetailLoaded(this.snippet);
}

class SnippetCreated extends SnippetState {
  final Snippet snippet;
  SnippetCreated(this.snippet);
}

class SnippetError extends SnippetState {
  final String message;
  SnippetError(this.message);
}

class SnippetActionSuccess extends SnippetState {
  final String message;
  SnippetActionSuccess(this.message);
}

// Bloc
class SnippetBloc extends Bloc<SnippetEvent, SnippetState> {
  final SnippetService _service;

  SnippetBloc({required SnippetService service})
      : _service = service,
        super(SnippetInitial()) {
    on<SnippetListRequested>(_onListRequested);
    on<SnippetLoadRequested>(_onLoadRequested);
    on<SnippetCreate>(_onCreate);
    on<SnippetUpdate>(_onUpdate);
    on<SnippetDelete>(_onDelete);
    on<SnippetSearchChanged>(_onSearchChanged);
  }

  Future<void> _onListRequested(SnippetListRequested event, Emitter<SnippetState> emit) async {
    emit(SnippetLoading());
    try {
      final snippets = await _service.getSnippets();
      emit(SnippetListLoaded(snippets));
    } catch (e) {
      emit(SnippetError(e.toString()));
    }
  }

  Future<void> _onLoadRequested(SnippetLoadRequested event, Emitter<SnippetState> emit) async {
    emit(SnippetLoading());
    try {
      final snippet = await _service.getSnippet(event.id);
      emit(SnippetDetailLoaded(snippet));
    } catch (e) {
      emit(SnippetError(e.toString()));
    }
  }

  Future<void> _onCreate(SnippetCreate event, Emitter<SnippetState> emit) async {
    try {
      final snippet = await _service.createSnippet(
        title: event.title,
        content: event.content,
        language: event.language,
      );
      emit(SnippetCreated(snippet));
      emit(SnippetActionSuccess('Snippet created'));
    } catch (e) {
      emit(SnippetError(e.toString()));
    }
  }

  Future<void> _onUpdate(SnippetUpdate event, Emitter<SnippetState> emit) async {
    try {
      final snippet = await _service.updateSnippet(
        event.id,
        title: event.title,
        content: event.content,
        language: event.language,
      );
      emit(SnippetDetailLoaded(snippet));
      emit(SnippetActionSuccess('Snippet updated'));
    } catch (e) {
      emit(SnippetError(e.toString()));
    }
  }

  Future<void> _onDelete(SnippetDelete event, Emitter<SnippetState> emit) async {
    try {
      await _service.deleteSnippet(event.id);
      emit(SnippetActionSuccess('Snippet deleted'));
      add(SnippetListRequested());
    } catch (e) {
      emit(SnippetError(e.toString()));
    }
  }

  Future<void> _onSearchChanged(SnippetSearchChanged event, Emitter<SnippetState> emit) async {
    if (state is SnippetListLoaded) {
      final current = state as SnippetListLoaded;
      emit(SnippetListLoaded(current.snippets, searchQuery: event.query));
    }
  }
}
