class Host {
  final String id;
  final String name;
  final String address;
  final int port;
  final String? username;
  final List<String> tags;
  final DateTime createdAt;

  Host({
    required this.id,
    required this.name,
    required this.address,
    this.port = 22,
    this.username,
    this.tags = const [],
    required this.createdAt,
  });

  // TODO: add fromJson, toJson
}
