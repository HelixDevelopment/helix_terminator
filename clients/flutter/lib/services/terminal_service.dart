import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';

/// Callback type for terminal messages received from the WebSocket.
typedef TerminalMessageCallback = void Function(String message);

/// Service that manages a WebSocket-based terminal connection.
class TerminalService {
  WebSocket? _socket;
  final _messageController = StreamController<String>.broadcast();
  TerminalMessageCallback? _onMessageCallback;
  String? _currentHostId;
  bool _isConnected = false;

  bool get isConnected => _isConnected;
  String? get currentHostId => _currentHostId;

  /// Establishes a WebSocket terminal connection for the given [hostId].
  Future<void> connect(String hostId) async {
    if (_isConnected) {
      await disconnect();
    }

    _currentHostId = hostId;
    final wsUrl = _resolveWsUrl(hostId);

    try {
      _socket = await WebSocket.connect(wsUrl);
      _isConnected = true;

      _socket!.listen(
        (data) {
          final message = data is String ? data : utf8.decode(data);
          _messageController.add(message);
          _onMessageCallback?.call(message);
        },
        onError: (error) {
          _messageController.add('\\x1b[31mConnection error: $error\\x1b[0m\\r\\n');
          _isConnected = false;
        },
        onDone: () {
          _isConnected = false;
          _messageController.add('\\x1b[33mConnection closed.\\x1b[0m\\r\\n');
        },
      );
    } catch (e) {
      _isConnected = false;
      throw Exception('Failed to connect to terminal: $e');
    }
  }

  /// Sends a raw command string to the terminal.
  Future<void> sendCommand(String command) async {
    if (_socket == null || _socket!.readyState != WebSocket.open) {
      throw Exception('Terminal not connected');
    }
    _socket!.add(command);
  }

  /// Closes the WebSocket connection.
  Future<void> disconnect() async {
    _isConnected = false;
    await _socket?.close();
    _socket = null;
    _currentHostId = null;
  }

  /// Registers a callback to be invoked on every incoming message.
  void onMessage(TerminalMessageCallback callback) {
    _onMessageCallback = callback;
  }

  /// Removes the message callback.
  void removeOnMessage() {
    _onMessageCallback = null;
  }

  /// Stream of incoming terminal messages.
  Stream<String> get messageStream => _messageController.stream;

  void dispose() {
    disconnect();
    _messageController.close();
  }

  String _resolveWsUrl(String hostId) {
    // In production this should read from environment / config.
    const base = 'wss://api.helix terminator.example.com';
    return '$base/v1/terminal/$hostId';
  }
}
