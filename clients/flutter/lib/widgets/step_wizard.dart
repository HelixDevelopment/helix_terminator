import 'package:flutter/material.dart';

class StepWizard extends StatelessWidget {
  final int currentStep;
  final List<String> steps;
  final Widget content;

  const StepWizard({super.key, required this.currentStep, required this.steps, required this.content});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Stepper(
          currentStep: currentStep,
          controlsBuilder: (_, __) => const SizedBox.shrink(),
          steps: steps.map((s) => Step(title: Text(s), content: const SizedBox.shrink())).toList(),
        ),
        Expanded(child: content),
      ],
    );
  }
}
