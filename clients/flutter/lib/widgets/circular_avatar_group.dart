import 'package:flutter/material.dart';

class CircularAvatarGroup extends StatelessWidget {
  final List<String> imageUrls;
  final double radius;

  const CircularAvatarGroup({super.key, required this.imageUrls, this.radius = 20});

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: -8,
      children: imageUrls.map((url) {
        return CircleAvatar(
          radius: radius,
          backgroundImage: NetworkImage(url),
        );
      }).toList(),
    );
  }
}
