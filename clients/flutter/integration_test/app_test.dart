import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:helix_terminator/main.dart' as app;

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  testWidgets('end-to-end test stub', (WidgetTester tester) async {
    app.main();
    await tester.pumpAndSettle();
    // TODO: implement real e2e tests
    expect(find.byType(MaterialApp), findsOneWidget);
  });
}
