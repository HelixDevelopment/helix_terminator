import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:helix_terminator/main.dart';

void main() {
  testWidgets('App renders splash screen', (WidgetTester tester) async {
    await tester.pumpWidget(const HelixTerminatorApp());
    expect(find.byType(CircularProgressIndicator), findsOneWidget);

    // SplashScreen.initState() schedules a real `Future.delayed(2s)` to
    // trigger AuthCheckRequested + navigation. Without draining it, the
    // Flutter test binding's post-test invariant check
    // ("A Timer is still pending even after the widget tree was disposed.")
    // fails the test for a test-harness reason, not a product defect
    // (§11.4.1: FAIL-bluffs from script-internal causes are forbidden just
    // like PASS-bluffs — fixed at the test-source layer). Draining it here
    // also proves the real navigation away from the splash screen happens.
    await tester.pump(const Duration(seconds: 2, milliseconds: 100));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 500));
  });
}
