class Secret {
  final String id;
  final String name;
  final String type; // password, key, token
  final DateTime createdAt;

  Secret({
    required this.id,
    required this.name,
    required this.type,
    required this.createdAt,
  });

  // TODO: add fromJson, toJson
}
