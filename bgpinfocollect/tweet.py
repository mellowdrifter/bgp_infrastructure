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


def update(deltaH: int, deltaW:int) -> str:
    if deltaH == 1:
        update = "This is 1 more prefix than 6 hours ago "
    elif deltaH == -1:
        update = "This is 1 less prefix than 6 hours ago "
    elif deltaH < 0:
        update = "This is " + str(-deltaH) + " fewer prefixes than 6 hours ago "
    elif deltaH > 0:
        update = "This is " + str(deltaH) + " more prefixes than 6 hours ago "
    else:
        update = "No change in the amount of prefixes from 6 hours ago "

    if deltaW == 1:
        update += "and 1 more than a week ago"
    elif deltaW == -1:
        update += "and 1 less than a week ago"
    elif deltaW < 0:
        update += "and " + str(-deltaW) + " fewer than a week ago"
    elif deltaW > 0:
        update += "and " + str(deltaW) + " more than a week ago"
    else:
        update += "and no change in the amount from a week ago"

    return update


def sendCount(dry: bool) -> (str, str):
    print('Running send counts')
    counts = stub.get_prefix_count(pb.empty())

    t4 = "I see " + str(counts.currentv4) + " IPv4 prefixes. "
    t6 = "I see " + str(counts.currentv6) + " IPv6 prefixes. "

    # Work out deltas
    v4_deltaH = counts.currentv4 - counts.sixhoursv4
    v6_deltaH = counts.currentv6 - counts.sixhoursv6
    v4_deltaW = counts.currentv4 - counts.weekagov4
    v6_deltaW = counts.currentv6 - counts.weekagov6

    t4 += update(v4_deltaH, v4_deltaW)
    t6 += update(v6_deltaH, v6_deltaW)

    # What percentage is taken up by /24 and /48?
    t4 += ". " + str(round(counts.slash24/float(counts.currentv4) * 100, 2)) + "% of prefixes are /24."
    t6 += ". " + str(round(counts.slash48/float(counts.currentv6) * 100, 2)) + "% of prefixes are /48."

    if dry:
        print("DRY RUN!!!")
    else:
        print(counts.time)
        stub.set_tweet_bit(pb.time_v4_v6(
            time = counts.time
        ))

    return t4, t6

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
    suffix = str(config.get('tweet', 'suffix'))
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


    dry = False

    if args.dry_run:
        print("will dry run")
        dry = True
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
        print(sendCount(dry))
    elif args.type == 'pie':
        sendPie()
    elif args.type == 'graph':
        if args.period not in validPeriod:
            print('Not a valid period for graph')
            sys.exit()
        sendGraph(args.period)

    # Done our work, get out of here
    sys.exit()
