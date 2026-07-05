import 'package:flutter/material.dart';

class BreadcrumbFilePath extends StatelessWidget {
  final List<String> segments;
  final ValueChanged<int>? onTapSegment;

  const BreadcrumbFilePath({super.key, required this.segments, this.onTapSegment});

  @override
  Widget build(BuildContext context) {
    return Wrap(
      children: [
        for (int i = 0; i < segments.length; i++) ...[
          GestureDetector(
            onTap: () => onTapSegment?.call(i),
            child: Text(segments[i]),
          ),
          if (i < segments.length - 1) const Text(' / '),
        ],
      ],
    );
  }
}
