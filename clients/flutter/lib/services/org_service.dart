import '../models/organization.dart';
import 'api_client.dart';

class OrgService {
  final ApiClient _apiClient;

  OrgService({required ApiClient apiClient}) : _apiClient = apiClient;

  Future<Organization> getOrganization() async {
    final response = await _apiClient.get('/api/v1/organization');
    return _organizationFromJson(response['data'] as Map<String, dynamic>);
  }

  Future<Organization> updateOrganization({
    String? name,
    String? slug,
    String? logoUrl,
  }) async {
    final body = <String, dynamic>{};
    if (name != null) body['name'] = name;
    if (slug != null) body['slug'] = slug;
    if (logoUrl != null) body['logoUrl'] = logoUrl;
    final response = await _apiClient.post('/api/v1/organization', body);
    return _organizationFromJson(response['data'] as Map<String, dynamic>);
  }

  Future<List<Map<String, dynamic>>> getMembers() async {
    final response = await _apiClient.get('/api/v1/organization/members');
    return (response['data'] as List<dynamic>? ?? [])
        .map((e) => e as Map<String, dynamic>)
        .toList();
  }

  Future<void> inviteMember(String email, String role) async {
    await _apiClient.post('/api/v1/organization/members', {
      'email': email,
      'role': role,
    });
  }

  Future<void> removeMember(String userId) async {
    await _apiClient.post('/api/v1/organization/members/$userId/remove', {});
  }

  Future<void> updateMemberRole(String userId, String role) async {
    await _apiClient.post('/api/v1/organization/members/$userId/role', {
      'role': role,
    });
  }

  Organization _organizationFromJson(Map<String, dynamic> json) {
    return Organization(
      id: json['id'] as String,
      name: json['name'] as String,
      slug: json['slug'] as String,
      logoUrl: json['logoUrl'] as String?,
      createdAt: DateTime.parse(json['createdAt'] as String),
    );
  }
}
