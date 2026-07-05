class PortForward {
  final String id;
  final String hostId;
  final int localPort;
  final int remotePort;
  final String remoteHost;
  final bool active;

  PortForward({
    required this.id,
    required this.hostId,
    required this.localPort,
    required this.remotePort,
    this.remoteHost = 'localhost',
    this.active = false,
  });

  // TODO: add fromJson, toJson
}
