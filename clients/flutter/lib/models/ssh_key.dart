class SshKey {
  final String id;
  final String name;
  final String fingerprint;
  final DateTime createdAt;

  SshKey({
    required this.id,
    required this.name,
    required this.fingerprint,
    required this.createdAt,
  });

  // TODO: add fromJson, toJson
}
