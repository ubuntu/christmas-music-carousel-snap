# -*- coding: utf-8 -*-

from __future__ import print_function
import logging

import argparse
LOGGER = logging.getLogger(__name__)


def main():
    """Main entry point of the program"""

    parser = argparse.ArgumentParser(description="Receive some midi events and forward in grpc PiGlow light commands")
    parser.add_argument('MidiPort', help="midi port to connect from")
    parser.add_argument('address', metavar='IP:PORT', help="grpc PiGlow IP:port to forward to")

    parser.add_argument("-d", "--debug", action="store_true", help="Debug mode")

    args = parser.parse_args()

    logging.basicConfig(level=logging.INFO, format="%(message)s")
    if args.debug:
        logging.basicConfig(level=logging.DEBUG, format="%(levelname)s: %(message)s")
        LOGGER.debug("Debug mode enabled")

    print(args)
