import 'package:flutter/material.dart';

class NetworkImageWithFallback extends StatelessWidget {
  final String url;
  final Widget fallback;

  const NetworkImageWithFallback({super.key, required this.url, required this.fallback});

  @override
  Widget build(BuildContext context) {
    return Image.network(
      url,
      errorBuilder: (_, __, ___) => fallback,
    );
  }
}
