class Workspace {
  final String id;
  final String name;
  final String? description;
  final List<String> hostIds;
  final List<String> memberIds;
  final DateTime createdAt;
  final DateTime? updatedAt;

  Workspace({
    required this.id,
    required this.name,
    this.description,
    this.hostIds = const [],
    this.memberIds = const [],
    required this.createdAt,
    this.updatedAt,
  });

  Workspace copyWith({
    String? id,
    String? name,
    String? description,
    List<String>? hostIds,
    List<String>? memberIds,
    DateTime? createdAt,
    DateTime? updatedAt,
  }) {
    return Workspace(
      id: id ?? this.id,
      name: name ?? this.name,
      description: description ?? this.description,
      hostIds: hostIds ?? this.hostIds,
      memberIds: memberIds ?? this.memberIds,
      createdAt: createdAt ?? this.createdAt,
      updatedAt: updatedAt ?? this.updatedAt,
    );
  }
}
