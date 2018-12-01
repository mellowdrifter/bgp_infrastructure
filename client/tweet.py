#!/usr/bin/env python3

import datetime
import configparser
import numpy as np
import sys
import matplotlib
matplotlib.use('Agg')
from matplotlib import pyplot as plt
import matplotlib.dates as mdates
from twython import Twython
import argparse
import os

# Each twitter account has their own keys
def accountKeys(family):
    config = configparser.ConfigParser()
    path = "{}/config.ini".format(os.path.dirname(os.path.realpath(__file__)))
    config.read(path)
    if family == 4:
        consumer_key = config.get('bgp4_account', 'consumer_key')
        consumer_secret = config.get('bgp4_account', 'consumer_secret')
        access_token = config.get('bgp4_account', 'access_token')
        access_token_secret = config.get('bgp4_account', 'access_token_secret')
    if family == 6:
        consumer_key = config.get('bgp6_account', 'consumer_key')
        consumer_secret = config.get('bgp6_account', 'consumer_secret')
        access_token = config.get('bgp6_account', 'access_token')
        access_token_secret = config.get('bgp6_account', 'access_token_secret')
    return consumer_key, consumer_secret, access_token, access_token_secret

def sendCount():
    print('send counts')

def sendPie():
    print('send pie')

def sendGraph(period):
    print('send graph for', period)

if __name__ == "__main__":
    validTypes = ['count', 'pie', 'graph']
    validPeriod = ['w', 'm', 's', 'a']
    parser = argparse.ArgumentParser(description='Formulate tweets and send to the world')
    parser.add_argument('-t', '--type', required=True, type=str, help='Type of tweet')
    parser.add_argument('-p', '--period', required=False)
    parser.add_argument('-d', '--dry_run', required=False)
    args = parser.parse_args()

    if args.dry_run:
        print("will dry run")
    else:
        print("Won't dry run")

    # only handle correct types
    if args.type not in validTypes:
        print('Not a valid type')
        sys.exit()

    if args.type == 'count':
        sendCount()
    elif args.type == 'pie':
        sendPie()
    elif args.type == 'graph':
        if args.period not in validPeriod:
            print('Not a valid period for graph')
            sys.exit()
        sendGraph(args.period)

    # Done our work, get out of here
    sys.exit()
