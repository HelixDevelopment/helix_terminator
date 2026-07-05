class Recording {
  final String id;
  final String sessionId;
  final String title;
  final Duration duration;
  final DateTime createdAt;

  Recording({
    required this.id,
    required this.sessionId,
    required this.title,
    required this.duration,
    required this.createdAt,
  });

  // TODO: add fromJson, toJson
}
