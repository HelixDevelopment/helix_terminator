class Workspace {
  final String id;
  final String name;
  final String? description;
  final List<String> hostIds;
  final DateTime createdAt;

  Workspace({
    required this.id,
    required this.name,
    this.description,
    this.hostIds = const [],
    required this.createdAt,
  });

  // TODO: add fromJson, toJson
}
