class Notification {
  final String id;
  final String title;
  final String body;
  final bool read;
  final DateTime createdAt;

  Notification({
    required this.id,
    required this.title,
    required this.body,
    this.read = false,
    required this.createdAt,
  });

  // TODO: add fromJson, toJson
}
