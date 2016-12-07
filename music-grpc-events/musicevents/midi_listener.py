# -*- coding: utf-8 -*-

from __future__ import print_function
from collections import namedtuple
import logging
from random import randint

import midi
import midi.sequencer as sequencer
import midi.sequencer.sequencer_alsa as S

LOGGER = logging.getLogger(__name__)

LedTracker = namedtuple('LedTracker', ['note', 'channel'])

class MidiSequencer(object):
    '''Midi sequencer transforming events to grpc piglow requests'''

    def __init__(self, midiport, piglow):
        '''Create a Midi listener sequencer'''
        LOGGER.debug("Connecting midi to port: " + midiport)
        client, port = midiport.split(':')

        self.seq = sequencer.SequencerRead(sequencer_resolution=120)
        self.seq.set_nonblock(False)
        self.seq.subscribe_port(client, port)
        self.seq.start_sequencer()

        self.piglow = piglow
        self.reset_leds_tracking()

    def reset_leds_tracking(self):
        """Recreate a new map array of leds"""
        self.leds = [None] * self.piglow.NUM_LEDS

    def listen(self):
        '''Listen mainloop and send event to piglow'''

        while True:
            ev = S.event_input(self.seq.client)
            if not ev:
                continue
            if (ev < 0):
                self.seq._error(ev)
                continue

            if ev.type == S.SND_SEQ_EVENT_PGMCHANGE:
                LOGGER.debug("New track")
                self.piglow.reset_leds()
                self.reset_leds_tracking()

            elif ev.type == S.SND_SEQ_EVENT_NOTEON and ev.data.note.velocity != 0:
                LOGGER.debug("New light on")
                channel = ev.data.note.channel
                if ((channel >= 8 and channel <= 15) or # percussions
                        (channel >= 32 and channel <= 39) or # base
                        (channel >= 88 and channel <= 103)): # synth
                    continue

                # get the list of free leds
                free_leds = []
                for i, l in enumerate(self.leds):
                    if not l:
                        free_leds.append(i)
                if not free_leds:
                    LOGGER.debug("No free leds to light up")
                    continue
                i = randint(0, len(free_leds) - 1)
                new_led_int = free_leds[i]

                # light up
                LOGGER.debug("Light on led #{}".format(new_led_int))
                self.piglow.set_led_on(new_led_int)

                # store this info
                self.leds[new_led_int] = LedTracker(ev.data.note.note, channel)

            elif (ev.type == S.SND_SEQ_EVENT_NOTEOFF or
                  (ev.type == S.SND_SEQ_EVENT_NOTEON and ev.data.note.velocity == 0)):
                LOGGER.debug("New light off")

                note, channel = ev.data.note.note, ev.data.note.channel
                for i, l in enumerate(self.leds):
                    if not l:
                        continue
                    if l.note == note and l.channel == channel:
                        LOGGER.debug("Light off led #{}".format(i))
                        self.piglow.set_led_off(i)
                        self.leds[i] = None
                        break

