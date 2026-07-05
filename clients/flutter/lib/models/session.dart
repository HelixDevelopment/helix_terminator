class Session {
  final String id;
  final String hostId;
  final DateTime startedAt;
  final DateTime? endedAt;
  final String protocol;

  Session({
    required this.id,
    required this.hostId,
    required this.startedAt,
    this.endedAt,
    this.protocol = 'ssh',
  });

  // TODO: add fromJson, toJson
}
