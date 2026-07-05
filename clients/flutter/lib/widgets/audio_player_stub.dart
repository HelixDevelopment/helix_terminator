import 'package:flutter/material.dart';

class AudioPlayerStub extends StatelessWidget {
  final String url;

  const AudioPlayerStub({super.key, required this.url});

  @override
  Widget build(BuildContext context) {
    // TODO: integrate just_audio
    return Row(
      children: [
        const Icon(Icons.play_arrow),
        Expanded(child: Text(url)),
      ],
    );
  }
}
