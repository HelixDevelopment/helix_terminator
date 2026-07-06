import 'api_client.dart';

class BillingService {
  final ApiClient _apiClient;

  BillingService({required ApiClient apiClient}) : _apiClient = apiClient;

  Future<Map<String, dynamic>> getCurrentPlan() async {
    final response = await _apiClient.get('/api/v1/billing/plan');
    return response['data'] as Map<String, dynamic>? ?? {};
  }

  Future<List<Map<String, dynamic>>> getInvoices() async {
    final response = await _apiClient.get('/api/v1/billing/invoices');
    return (response['data'] as List<dynamic>? ?? [])
        .map((e) => e as Map<String, dynamic>)
        .toList();
  }

  Future<Map<String, dynamic>> getUsageStats() async {
    final response = await _apiClient.get('/api/v1/billing/usage');
    return response['data'] as Map<String, dynamic>? ?? {};
  }

  Future<Map<String, dynamic>> getPaymentMethod() async {
    final response = await _apiClient.get('/api/v1/billing/payment-method');
    return response['data'] as Map<String, dynamic>? ?? {};
  }

  Future<void> updatePaymentMethod(Map<String, dynamic> paymentMethod) async {
    await _apiClient.post('/api/v1/billing/payment-method', paymentMethod);
  }

  Future<void> cancelSubscription() async {
    await _apiClient.post('/api/v1/billing/cancel', {});
  }

  Future<List<Map<String, dynamic>>> getAvailablePlans() async {
    final response = await _apiClient.get('/api/v1/billing/plans');
    return (response['data'] as List<dynamic>? ?? [])
        .map((e) => e as Map<String, dynamic>)
        .toList();
  }
}
