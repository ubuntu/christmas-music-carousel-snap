# christmas-music-carousel-snap
Play a christmas music carousel from a selection or pre-selected music. Can connect to grpc-piglow snap on a raspberry pi.

## Regenerating the gRPC protocol (python)
Ensure you are in your python virtualenv in music-grpc-events:
```
cd music-grpc-events
virtualenv venv
. venv/bin/active
pip install -r requirements.txt
```

Then, regenerate *piglow_pb2.py* from our protobuf protocole (the proto file is
in github.com/didrocks/grpc-piglow):

```
pip install grpcio-tools
python -m grpc.tools.protoc -Ipath_to_grpc-piglow/proto/ --python_out=musicevents/ --grpc_python_out=musicevents/ path_to_grpc-piglow/proto/piglow.proto
```
