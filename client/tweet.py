#!/usr/bin/env python3

import configparser
import os
from twython import Twython


class Message:
    def add_message(self, message):
        self.message = message
    def add_image(self, image):
        self.image = image
    def add_account(self, account):
        self.account = account
    def send_tweet(self):
        config = configparser.ConfigParser()
        path = "{}/config.ini".format(os.path.dirname(os.path.realpath(__file__)))
        config.read(path)
        consumer_key = config.get(self.account, 'consumer_key')
        consumer_secret = config.get(self.account, 'consumer_secret')
        access_token = config.get(self.account, 'access_token')
        access_token_secret = config.get(self.account, 'access_token_secret')

        twitter = Twython(consumer_key, consumer_secret,
            access_token, access_token_secret)
