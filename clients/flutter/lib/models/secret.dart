class Secret {
  final String id;
  final String name;
  final String type; // password, key, token
  final String category;
  final String? description;
  final DateTime createdAt;
  final DateTime? updatedAt;

  Secret({
    required this.id,
    required this.name,
    required this.type,
    this.category = 'general',
    this.description,
    required this.createdAt,
    this.updatedAt,
  });

  Secret copyWith({
    String? id,
    String? name,
    String? type,
    String? category,
    String? description,
    DateTime? createdAt,
    DateTime? updatedAt,
  }) {
    return Secret(
      id: id ?? this.id,
      name: name ?? this.name,
      type: type ?? this.type,
      category: category ?? this.category,
      description: description ?? this.description,
      createdAt: createdAt ?? this.createdAt,
      updatedAt: updatedAt ?? this.updatedAt,
    );
  }
}
