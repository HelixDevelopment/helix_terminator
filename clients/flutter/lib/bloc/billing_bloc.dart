import 'package:flutter_bloc/flutter_bloc.dart';
import '../services/billing_service.dart';

// Events
abstract class BillingEvent {}

class BillingDashboardRequested extends BillingEvent {}

class BillingUpdatePaymentMethod extends BillingEvent {
  final Map<String, dynamic> paymentMethod;
  BillingUpdatePaymentMethod(this.paymentMethod);
}

class BillingCancelSubscription extends BillingEvent {}

class BillingSelectPlan extends BillingEvent {
  final String planId;
  BillingSelectPlan(this.planId);
}

// States
abstract class BillingState {}

class BillingInitial extends BillingState {}

class BillingLoading extends BillingState {}

class BillingDashboardLoaded extends BillingState {
  final Map<String, dynamic> plan;
  final List<Map<String, dynamic>> invoices;
  final Map<String, dynamic> usage;
  final Map<String, dynamic> paymentMethod;
  BillingDashboardLoaded({
    required this.plan,
    required this.invoices,
    required this.usage,
    required this.paymentMethod,
  });
}

class BillingError extends BillingState {
  final String message;
  BillingError(this.message);
}

class BillingActionSuccess extends BillingState {
  final String message;
  BillingActionSuccess(this.message);
}

// Bloc
class BillingBloc extends Bloc<BillingEvent, BillingState> {
  final BillingService _service;

  BillingBloc({required BillingService service})
      : _service = service,
        super(BillingInitial()) {
    on<BillingDashboardRequested>(_onDashboardRequested);
    on<BillingUpdatePaymentMethod>(_onUpdatePaymentMethod);
    on<BillingCancelSubscription>(_onCancelSubscription);
    on<BillingSelectPlan>(_onSelectPlan);
  }

  Future<void> _onDashboardRequested(BillingDashboardRequested event, Emitter<BillingState> emit) async {
    emit(BillingLoading());
    try {
      final plan = await _service.getCurrentPlan();
      final invoices = await _service.getInvoices();
      final usage = await _service.getUsageStats();
      final paymentMethod = await _service.getPaymentMethod();
      emit(BillingDashboardLoaded(
        plan: plan,
        invoices: invoices,
        usage: usage,
        paymentMethod: paymentMethod,
      ));
    } catch (e) {
      emit(BillingError(e.toString()));
    }
  }

  Future<void> _onUpdatePaymentMethod(BillingUpdatePaymentMethod event, Emitter<BillingState> emit) async {
    try {
      await _service.updatePaymentMethod(event.paymentMethod);
      emit(BillingActionSuccess('Payment method updated'));
      add(BillingDashboardRequested());
    } catch (e) {
      emit(BillingError(e.toString()));
    }
  }

  Future<void> _onCancelSubscription(BillingCancelSubscription event, Emitter<BillingState> emit) async {
    try {
      await _service.cancelSubscription();
      emit(BillingActionSuccess('Subscription cancelled'));
      add(BillingDashboardRequested());
    } catch (e) {
      emit(BillingError(e.toString()));
    }
  }

  Future<void> _onSelectPlan(BillingSelectPlan event, Emitter<BillingState> emit) async {
    try {
      emit(BillingActionSuccess('Plan selected'));
      add(BillingDashboardRequested());
    } catch (e) {
      emit(BillingError(e.toString()));
    }
  }
}
