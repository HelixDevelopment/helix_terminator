import 'package:flutter_bloc/flutter_bloc.dart';
import '../models/organization.dart';
import '../services/org_service.dart';

// Events
abstract class OrgEvent {}

class OrgDashboardRequested extends OrgEvent {}

class OrgInviteMember extends OrgEvent {
  final String email;
  final String role;
  OrgInviteMember(this.email, this.role);
}

class OrgRemoveMember extends OrgEvent {
  final String userId;
  OrgRemoveMember(this.userId);
}

class OrgUpdateMemberRole extends OrgEvent {
  final String userId;
  final String role;
  OrgUpdateMemberRole(this.userId, this.role);
}

class OrgUpdateSettings extends OrgEvent {
  final String? name;
  final String? slug;
  final String? logoUrl;
  OrgUpdateSettings({this.name, this.slug, this.logoUrl});
}

// States
abstract class OrgState {}

class OrgInitial extends OrgState {}

class OrgLoading extends OrgState {}

class OrgDashboardLoaded extends OrgState {
  final Organization? organization;
  final List<Map<String, dynamic>> members;
  OrgDashboardLoaded({this.organization, required this.members});
}

class OrgError extends OrgState {
  final String message;
  OrgError(this.message);
}

class OrgActionSuccess extends OrgState {
  final String message;
  OrgActionSuccess(this.message);
}

// Bloc
class OrgBloc extends Bloc<OrgEvent, OrgState> {
  final OrgService _service;

  OrgBloc({required OrgService service})
      : _service = service,
        super(OrgInitial()) {
    on<OrgDashboardRequested>(_onDashboardRequested);
    on<OrgInviteMember>(_onInviteMember);
    on<OrgRemoveMember>(_onRemoveMember);
    on<OrgUpdateMemberRole>(_onUpdateMemberRole);
    on<OrgUpdateSettings>(_onUpdateSettings);
  }

  Future<void> _onDashboardRequested(OrgDashboardRequested event, Emitter<OrgState> emit) async {
    emit(OrgLoading());
    try {
      final organization = await _service.getOrganization();
      final members = await _service.getMembers();
      emit(OrgDashboardLoaded(organization: organization, members: members));
    } catch (e) {
      emit(OrgError(e.toString()));
    }
  }

  Future<void> _onInviteMember(OrgInviteMember event, Emitter<OrgState> emit) async {
    try {
      await _service.inviteMember(event.email, event.role);
      emit(OrgActionSuccess('Member invited'));
      add(OrgDashboardRequested());
    } catch (e) {
      emit(OrgError(e.toString()));
    }
  }

  Future<void> _onRemoveMember(OrgRemoveMember event, Emitter<OrgState> emit) async {
    try {
      await _service.removeMember(event.userId);
      emit(OrgActionSuccess('Member removed'));
      add(OrgDashboardRequested());
    } catch (e) {
      emit(OrgError(e.toString()));
    }
  }

  Future<void> _onUpdateMemberRole(OrgUpdateMemberRole event, Emitter<OrgState> emit) async {
    try {
      await _service.updateMemberRole(event.userId, event.role);
      emit(OrgActionSuccess('Role updated'));
      add(OrgDashboardRequested());
    } catch (e) {
      emit(OrgError(e.toString()));
    }
  }

  Future<void> _onUpdateSettings(OrgUpdateSettings event, Emitter<OrgState> emit) async {
    try {
      await _service.updateOrganization(name: event.name, slug: event.slug, logoUrl: event.logoUrl);
      emit(OrgActionSuccess('Settings updated'));
      add(OrgDashboardRequested());
    } catch (e) {
      emit(OrgError(e.toString()));
    }
  }
}
