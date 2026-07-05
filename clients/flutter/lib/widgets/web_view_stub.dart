import 'package:flutter/material.dart';

class WebViewStub extends StatelessWidget {
  final String url;

  const WebViewStub({super.key, required this.url});

  @override
  Widget build(BuildContext context) {
    // TODO: replace with webview_flutter or webview_windows
    return Center(child: Text('WebView: $url'));
  }
}
