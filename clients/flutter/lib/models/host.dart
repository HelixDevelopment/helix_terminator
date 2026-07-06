class Host {
  final String id;
  final String name;
  final String address;
  final int port;
  final String? username;
  final List<String> tags;
  final DateTime createdAt;
  final String status;
  final String? organizationId;
  final String authMethod;

  Host({
    required this.id,
    required this.name,
    required this.address,
    this.port = 22,
    this.username,
    this.tags = const [],
    required this.createdAt,
    this.status = 'unknown',
    this.organizationId,
    this.authMethod = 'password',
  });

  Host copyWith({
    String? id,
    String? name,
    String? address,
    int? port,
    String? username,
    List<String>? tags,
    DateTime? createdAt,
    String? status,
    String? organizationId,
    String? authMethod,
  }) {
    return Host(
      id: id ?? this.id,
      name: name ?? this.name,
      address: address ?? this.address,
      port: port ?? this.port,
      username: username ?? this.username,
      tags: tags ?? this.tags,
      createdAt: createdAt ?? this.createdAt,
      status: status ?? this.status,
      organizationId: organizationId ?? this.organizationId,
      authMethod: authMethod ?? this.authMethod,
    );
  }

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Host &&
          runtimeType == other.runtimeType &&
          id == other.id &&
          name == other.name &&
          address == other.address &&
          port == other.port &&
          username == other.username &&
          tags == other.tags &&
          status == other.status &&
          organizationId == other.organizationId &&
          authMethod == other.authMethod;

  @override
  int get hashCode =>
      id.hashCode ^
      name.hashCode ^
      address.hashCode ^
      port.hashCode ^
      username.hashCode ^
      tags.hashCode ^
      status.hashCode ^
      organizationId.hashCode ^
      authMethod.hashCode;
}
