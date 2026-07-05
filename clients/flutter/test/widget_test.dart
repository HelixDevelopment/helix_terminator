import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:helix_terminator/main.dart';

void main() {
  testWidgets('App renders splash screen', (WidgetTester tester) async {
    await tester.pumpWidget(const HelixTerminatorApp());
    expect(find.byType(CircularProgressIndicator), findsOneWidget);
  });
}
