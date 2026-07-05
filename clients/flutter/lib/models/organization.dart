class Organization {
  final String id;
  final String name;
  final String slug;
  final String? logoUrl;
  final DateTime createdAt;

  Organization({
    required this.id,
    required this.name,
    required this.slug,
    this.logoUrl,
    required this.createdAt,
  });

  // TODO: add fromJson, toJson
}
