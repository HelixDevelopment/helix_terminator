import 'package:flutter_bloc/flutter_bloc.dart';

// TODO: define collaboration events and states

class CollaborationBloc extends Bloc<CollaborationEvent, CollaborationState> {
  CollaborationBloc() : super(CollaborationInitial()) {
    on<CollaborationSessionStarted>((event, emit) {
      emit(CollaborationActive());
    });
  }
}

abstract class CollaborationEvent {}

class CollaborationSessionStarted extends CollaborationEvent {}

abstract class CollaborationState {}

class CollaborationInitial extends CollaborationState {}
class CollaborationActive extends CollaborationState {}
