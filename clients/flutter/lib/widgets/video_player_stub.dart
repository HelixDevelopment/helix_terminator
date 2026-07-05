import 'package:flutter/material.dart';

class VideoPlayerStub extends StatelessWidget {
  final String url;

  const VideoPlayerStub({super.key, required this.url});

  @override
  Widget build(BuildContext context) {
    // TODO: integrate video_player
    return Container(
      color: Colors.black,
      child: Center(child: Text('Video: $url', style: const TextStyle(color: Colors.white))),
    );
  }
}
