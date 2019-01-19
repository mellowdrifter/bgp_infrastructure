#!/usr/bin/env python3
"""
tweet.py is used to interact with twitter. It will use
the correct account details and send the requested messages.

It also creates the visual graphs using matplotlib.
"""


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
import bgpinfo_pb2 as pb
import bgpinfo_pb2_grpc
import grpc


def sendCount():
    print('Running send counts')
    result = stub.get_prefix_count(pb.empty())
    print(result)

def sendPie():
    print('send pie')

def sendGraph(period: str):
    print('send graph for', period)
    req = pb.length()
    if period == 'w':
        req.time = pb.WEEK
    if period == 'm':
        req.time = pb.MONTH
    if period == 's':
        req.time = pb.SIXMONTH
    if period == 'y':
        req.time = pb.YEAR
    result = stub.get_graph_data(req)
    print(len(result.tick))

def tweet(message, image, family, dryRun):
    if dryRun:
        print(message)
        return


# Each twitter account has their own keys
def accountKeys(family: int):
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


# main figures out which tweet is being requested and does
# sanity checking on the request.
if __name__ == "__main__":
    validTypes = ['count', 'pie', 'graph']
    validPeriod = ['w', 'm', 's', 'y']
    parser = argparse.ArgumentParser(description='Formulate tweets and send to the world')
    parser.add_argument('-t', '--type', required=True, type=str, help='Type of tweet')
    parser.add_argument('-p', '--period', required=False)
    parser.add_argument('-d', '--dry_run', required=False)
    args = parser.parse_args()

    if args.dry_run:
        print("will dry run")
    else:
        print("Won't dry run")

    # Load config
    config = configparser.ConfigParser()
    path = "{}/config.ini".format(os.path.dirname(os.path.realpath(__file__)))
    config.read(path)
    server = str(config.get('grpc', 'server'))
    port = str(config.get('grpc', 'port'))

    #TODO: SET UP LOGGING

    #Set up GRPC server details
    grpcserver = "%s:%s" % (server, port)
    channel = grpc.insecure_channel(grpcserver)
    stub = bgpinfo_pb2_grpc.bgp_infoStub(channel)

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
