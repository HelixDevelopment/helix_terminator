import 'package:flutter/material.dart';

class MapViewStub extends StatelessWidget {
  final double lat;
  final double lng;

  const MapViewStub({super.key, required this.lat, required this.lng});

  @override
  Widget build(BuildContext context) {
    // TODO: integrate flutter_map or google_maps_flutter
    return Container(
      color: Colors.grey.shade300,
      child: Center(child: Text('Map: $lat, $lng')),
    );
  }
}
