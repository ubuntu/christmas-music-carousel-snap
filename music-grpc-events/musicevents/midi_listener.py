# -*- coding: utf-8 -*-

from __future__ import print_function
import logging

import midi
import midi.sequencer as sequencer
import midi.sequencer.sequencer_alsa as S

LOGGER = logging.getLogger(__name__)

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

    def listen(self):
        '''Listen mainloop and send event to piglow'''

        while True:
            ev = S.event_input(self.seq.client)
            if ev:
                if (ev < 0):
                    self.seq._error(ev)
                    continue
                if ev.type == S.SND_SEQ_EVENT_PGMCHANGE:
                    LOGGER.debug("New track")
                    self.piglow.reset_leds()
                    # TODO: set channels instruments
                elif ev.type in (S.SND_SEQ_EVENT_NOTEON, S.SND_SEQ_EVENT_NOTEOFF):
                    print(ev)

