import logging

import grpc
import piglow_pb2 as pb

LOGGER = logging.getLogger(__name__)

class RemotePiGlow(object):
    '''Remote piGlow representation'''

    NUM_LEDS = 18
    MAX_BRIGHTNESS = 20

    def __init__(self, address):
        '''connect to PiGlow'''
        # Connect to grpc client
        LOGGER.debug("Connecting to PiGlow")
        channel = grpc.insecure_channel(address)
        self._piglow = pb.PiGlowStub(channel)

        # test connexion by zeroIng the leds
        self.reset_leds()
        LOGGER.debug("Connected to PiGlow")

    def set_led_on(self, num):
        '''Set led num to max brightness'''
        self.set_led(num, self.MAX_BRIGHTNESS)

    def set_led_off(self, num):
        '''Shut led num to 0'''
        self.set_led(num, 0)

    def set_led(self, num, brightness):
        '''Set remote led num to brightness'''
        LOGGER.debug("Set led {} to {}".format(num, brightness))
        self._piglow.SetLED(pb.LedRequest(num=num, brightness=brightness))

    def reset_leds(self):
        '''Set all leds brightness off'''
        LOGGER.debug("Setting all leds off")
        self._piglow.SetAll(pb.BrightnessRequest(brightness=0))
