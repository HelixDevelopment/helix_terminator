import 'package:flutter/material.dart';

class StepperScale extends StatelessWidget {
  const StepperScale({super.key});

  @override
  Widget build(BuildContext context) {
    return Stepper(
      currentStep: 0,
      steps: const [
        Step(title: Text('Step 1'), content: Text('Content 1')),
        Step(title: Text('Step 2'), content: Text('Content 2')),
        Step(title: Text('Step 3'), content: Text('Content 3')),
      ],
    );
  }
}
