import 'package:flutter/material.dart';

class ErrorBoundary extends StatefulWidget {
  final Widget child;

  const ErrorBoundary({super.key, required this.child});

  @override
  State<ErrorBoundary> createState() => _ErrorBoundaryState();
}

class _ErrorBoundaryState extends State<ErrorBoundary> {
  bool _hasError = false;

  @override
  Widget build(BuildContext context) {
    if (_hasError) {
      return const Center(child: Text('Something went wrong.'));
    }
    return widget.child;
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    // TODO: integrate with FlutterError.onError for real error boundaries
  }
}
