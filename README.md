# christmas-music-carousel-snap
Play a christmas music carousel from a selection or pre-selected music. Can connect to grpc-piglow snap on a raspberry pi.

## Setting up

On a 16.04 Ubuntu desktop, you can install this as a snap:

`snap install christmas-music-carousel --beta --devmode`

Optionally, connect on your network a Raspberry PiGlow
[with Ubuntu Core installed on it](https://developer.ubuntu.com/en/snappy/start/raspberry-pi-2/) with a PiGlow.
Install the grpc-piglow snap on it:

`snap install grpc-piglow --beta --devmode`

Finally, run the carousel with default music selection (as root):

`sudo christmas-music-carousel`

## Usage:
christmas-music-carousel [-options] [LIST OF MIDI FILES]

Play a music carousel and optionally sync up with lights on a Raspberry PiGlow
connected on the network.

A list of midi files can be provided, and in that case, the carousel will play
over them in random orders. If none is provided, a default christmas selection
is chosen.
If you have a PiGlow on the same network, ensure you have the grpc-piglow snap
installed on it.

This programs need to be ran as root on your laptop to connect to alsa.

## Available options

* brightness (integer): adjust brightness (from 1 to 255) for light up PiGlow. Warning: any value above default (30)
is dazzling.
* debug: Enable debug (developer) messages

## Technical notes:

This project is orchestrating multiple binaries:
* [TiMidity](http://timidity.sourceforge.net/), a software synthesizer playing MIDI files
* [alsa](http://www.alsa-project.org) utilities: aconnect, aplaymidi: to play midi files and connecting MIDI ports
* *music-grpc-events* a python program in **music-grpc-events/** directory receiving MIDI events and forwarding them to
the PiGlow RPI board via gRPC.
* *christmas-music-carousel*, a golang program in **chrismas-music-carousel* directory, which orchestrates all above
tools, restarting them as needed, handling the notions of required and optional components. It's using the bonjour/mDNS
protocol to detect the RPI PiGlow on the network and foward those connexions info to *music-grpc-events*

### Related projects

More information on gRPC-PiGlow can be found on https://github.com/didrocks/grpc-piglow


### Developing:

You need a Golang compiler with correct GOPATH set, and a python2 vm.

You also need timidity and freepats installed on your system. Ensure you don't have timidity running as a daemon.
`timidity-daemon` package shouldn't be installed or it will start a blocking timidity daemon locking alsa if your
hw doesn't handle multiple streams:
```
sudo apt install --no-install-recommends timidity freepats
```

Ensure you created a python virtualenv for music-grpc-events:
```
cd music-grpc-events/
virtualenv venv
. venv/bin/active
pip install -r requirements.txt
```

#### Regenerating the gRPC protocol (python)

In your virtualenv environment for music-grpc-events, regenerate *piglow_pb2.py* from our protobuf protocol (the proto file is
in github.com/didrocks/grpc-piglow):

```
pip install grpcio-tools
python -m grpc.tools.protoc -Ipath_to_grpc-piglow/proto/ --python_out=musicevents/ --grpc_python_out=musicevents/ path_to_grpc-piglow/proto/piglow.proto
```
