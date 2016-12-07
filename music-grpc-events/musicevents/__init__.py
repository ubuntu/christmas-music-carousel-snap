# -*- coding: utf-8 -*-

from __future__ import print_function
import logging
import signal
import sys

import argparse

import grpc_piglow
import midi_listener

LOGGER = logging.getLogger(__name__)

_piglow = None

def signal_handler(signal, frame):
    LOGGER.debug("Exit requested")
    if _piglow:
        _piglow.reset_leds()
    sys.exit(0)


def main():
    """Main entry point of the program"""

    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)

    parser = argparse.ArgumentParser(description="Receive some midi events and forward in grpc PiGlow light commands")
    parser.add_argument('MidiPort', help="midi port to connect from")
    parser.add_argument('address', metavar='IP:PORT', help="grpc PiGlow IP:port to forward to")
    parser.add_argument('-b', '--brightness', type=int, help="adjust brightness (from 1 to 255) for light up PiGlow. Warning: any value above default (30) is dazzling.")

    parser.add_argument("-d", "--debug", action="store_true", help="Debug mode")

    args = parser.parse_args()

    if args.debug:
        logging.basicConfig(level=logging.DEBUG, format="%(levelname)s: %(message)s")
        logging.debug("Debug mode enabled")
    else:
        logging.basicConfig(level=logging.INFO, format="%(message)s")

    b = args.brightness
    if b:
        if b < 1 or b > 255:
            LOGGER.debug("Keeping brightness default: value should be between 1 and 255. Got {}".format(b))
        else:
            grpc_piglow.RemotePiGlow.MAX_BRIGHTNESS = b

    # test connexion by zeroIng the leds
    try:
        piglow = grpc_piglow.RemotePiGlow(args.address)
        global _piglow
        _piglow = piglow
    except Exception as e:
        LOGGER.error("Couldn't connect to the PiGlow at " + args.address)
        print(e)
        sys.exit(1)

    # try to connect to midi sequencer
    try:
        seq = midi_listener.MidiSequencer(args.MidiPort, piglow)
    except ValueError:
        LOGGER.error("Invalid midi port parameter (should be client:port): " + args.MidiPort)
        sys.exit(1)
    except Exception as e:
        LOGGER.error("Couldn't connect midi sequencer:")
        print(e)
        sys.exit(1)

    seq.listen()
