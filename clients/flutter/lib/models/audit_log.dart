class AuditLog {
  final String id;
  final String actor;
  final String action;
  final String target;
  final DateTime timestamp;

  AuditLog({
    required this.id,
    required this.actor,
    required this.action,
    required this.target,
    required this.timestamp,
  });

  // TODO: add fromJson, toJson
}
