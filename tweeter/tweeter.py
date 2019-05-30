#!/usr/bin/env python3

import argparse
import bgpinfo_pb2 as pb
import bgpinfo_pb2_grpc
import configparser
import datetime
import grpc
import io
import logging
import matplotlib
matplotlib.use('Agg')
from matplotlib import pyplot as plt
from matplotlib import dates as mdates
import os
import sys
from twython import Twython
from typing import Tuple

# Load config
config = configparser.ConfigParser()
path = "{}/config.ini".format(os.path.dirname(os.path.realpath(__file__)))
config.read(path)
server = str(config.get('grpc', 'server'))
port = str(config.get('grpc', 'port'))
log = config.get('grpc', 'logfile')

# Check arguments
parser = argparse.ArgumentParser()
parser.add_argument("-dryrun", help="dryrun will read, but not tweet", default="True")
parser.add_argument("-call", help="getcurrent, getmovement, getpie, getrpki", default = "")
parser.add_argument("-time", help="week, month, month6, year", default="")
args = parser.parse_args()

# Set up logging
format_string = '%(levelname)s: %(asctime)s: %(message)s'
logging.basicConfig(filename=log, level=logging.INFO, format=format_string)

# Set up GRPC server details
grpcserver = "%s:%s" % (server, port)
channel = grpc.insecure_channel(grpcserver)
stub = bgpinfo_pb2_grpc.bgp_infoStub(channel)

# Time based variables
today = datetime.date.today()
yesterday = today - datetime.timedelta(days=1)
today = today.strftime("%d-%b-%Y")
yesterday = yesterday.strftime("%d-%b-%Y")

copyright = "data by: @mellowdrifter | www.mellowd.dev"


def getCurrent():
    """Grabs current v4 and v6 table count.
    This function will grab the current
    v4 and v6 count to tweet.

    requires:
    - current count
    - count from 6 hours ago
    - count from a week ago
    """
    logging.info('running getCurrent')
    result = stub.get_prefix_count(pb.empty())
    
    # Calculate deltas
    ipv4_deltaH = result.active_4 - result.sixhoursv4
    ipv6_deltaH = result.active_6 - result.sixhoursv6
    ipv4_deltaW = result.active_4 - result.weekagov4
    ipv6_deltaW = result.active_6 - result.weekagov6

    # Calculate large subnet percentages
    percent_v4 = (round(result.slash24 / float(result.active_4)*100, 2))
    percent_v6 = (round(result.slash48 / float(result.active_6)*100, 2))

    # Formulate update
    delta4 = "I see " + str(result.active_4) + " IPv4 prefixes" + ". "
    delta6 = "I see " + str(result.active_6) + " IPv6 prefixes" + ". "
    delta4 += createMessage(ipv4_deltaH, ipv4_deltaW)
    delta6 += createMessage(ipv6_deltaH, ipv6_deltaW)
    delta4 += '. ' + str(percent_v4) + '% of prefixes are /24.'
    delta6 += '. ' + str(percent_v6) + '% of prefixes are /48.'

    tweet(4, delta4, None)
    tweet(6, delta6, None)
    setTweetBit(result.time)


def getMovement(
    time_period: int,
    ):
    """Grabs weekly data for tweet.
    This function will grab the v4 and v6 
    counts over the last week.
    """
    # TODO fix descripions
    logging.info('running getMovement')
    message = {
        pb.movement_request.WEEK: "Weekly BGP table movement",
        pb.movement_request.MONTH: "Monthly BGP table movement",
        pb.movement_request.SIXMONTH: "BGP table movement for the last 6 months",
        pb.movement_request.ANNUAL: "Annual BGP table movement",
    }
    req = pb.movement_request()
    req.period = time_period
    result = stub.get_movement_totals(req)
    v4, v6 = createPlotGraph(result, time_period)
    tweet(4, message[time_period], v4)
    tweet(6, message[time_period], v6)

def getPrefixPie():
    """Create Pie graph.
    Grab the latest subnet size count and
    create pie graphs with those counts
    for each address family
    requires:
    - current spread of all subnet sizes.
    """
    logging.info('running getPrefixPie')
    result = stub.get_pie_subnets(pb.empty())
    v4, v6 = createPieGraph(result)
    tweet(4, "Current Prefix Distribution IPv4", v4)
    tweet(6, "Current Prefix Distribution IPv6", v6)

def getRPKIPie():
    """Create RPKI Pie graph.
    Grab the latest RPKI values and create
    pie graphs for each address family.
    requires:
    - current counts for all RPKI values.
    """
    logging.info('running getRPKIPie')
    result = stub.get_pie_rpki(pb.empty())
    v4, v6 = createRPKIPie(result)
    tweet(4, "Current RPKI status IPv4 #RPKI", v4)
    tweet(6, "Current RPKI status IPv6 #RPKI", v6)

def setTweetBit(time: int):
    """Set tweet bit.
    Updates database to show the latest tweeted
    values. Useful when comparing historically.
    """
    logging.info('running setTweetBit')
    if args.dryrun == "True":
        logging.info("Will set tweet bit with time {}".format(time))
        return
    else:
        timestamp = pb.timestamp()
        timestamp.time = time
        result = stub.update_tweet_bit(timestamp)
        logging.info("Tweet bit updated: {}".format(result.success))
    
def createPlotGraph(
    entries: pb.movement_totals_response,
    time_period: int,
    ) -> Tuple[io.BytesIO, io.BytesIO]:
    """Creates a plotted graph.
    Uses entries and time_period to create a
    matplotlib-based graph for the respective
    address family.
    """
    logging.info('running createPlotGraph')
    updates = {
        pb.movement_request.WEEK: "week",
        pb.movement_request.MONTH: "month",
        pb.movement_request.SIXMONTH: "6 months",
        pb.movement_request.ANNUAL: "year",
    }

    dates = []
    v4_counts = []
    v6_counts = []
    for values in entries.values:
        v4_counts.append(values.v4_values)
        v6_counts.append(values.v6_values)
        dates.append(datetime.datetime.fromtimestamp(values.time))
    
    # Start with the IPv4 graph
    plt.figure(figsize=(12, 10))
    ax = plt.subplot(111)
    xfmt = mdates.DateFormatter('%Y-%m-%d')
    ax.xaxis.set_major_formatter(xfmt)
    title = 'IPv4 table movement for {} ending {}'.format(
        updates[time_period], yesterday)
    plt.suptitle(title, fontsize=17)
    ax.grid(True)
    ax.spines["top"].set_visible(False)
    ax.spines["bottom"].set_visible(False)
    ax.spines["right"].set_visible(False)
    ax.spines["left"].set_visible(False)
    ax.get_xaxis().tick_bottom()
    ax.get_yaxis().tick_left()
    plt.xticks(fontsize=12, rotation=12)
    plt.yticks(fontsize=12)
    plt.ticklabel_format(axis='y', style='plain', useOffset=False)
    plt.tick_params(axis="both", which="both", bottom=False, top=False,
                    labelbottom=True, left=False, right=False, labelleft=True)
    plt.plot(dates, v4_counts, 'o-', lw=1, alpha=0.4, color="#238341")
    plt.figtext(0.5, 0.93, copyright,
                fontsize=14, color='gray', ha='center', va='top', alpha=0.8)

    v4graph = io.BytesIO()
    plt.savefig(v4graph, format='png')
    plt.close()

    # Now the IPv6 graph
    plt.figure(figsize=(12, 10))
    ax = plt.subplot(111)
    xfmt = mdates.DateFormatter('%Y-%m-%d')
    ax.xaxis.set_major_formatter(xfmt)
    title = 'IPv6 table movement for {} ending {}'.format(
        updates[time_period], yesterday)
    plt.suptitle(title, fontsize=17)
    ax.grid(True)
    ax.spines["top"].set_visible(False)
    ax.spines["bottom"].set_visible(False)
    ax.spines["right"].set_visible(False)
    ax.spines["left"].set_visible(False)
    ax.get_xaxis().tick_bottom()
    ax.get_yaxis().tick_left()
    plt.xticks(fontsize=12, rotation=12)
    plt.yticks(fontsize=12)
    plt.ticklabel_format(axis='y', style='plain', useOffset=False)
    plt.tick_params(axis="both", which="both", bottom=False, top=False,
                    labelbottom=True, left=False, right=False, labelleft=True)
    plt.plot(dates, v6_counts, 'o-', lw=1, alpha=0.4, color="#0041A0")
    plt.figtext(0.5, 0.93, copyright,
                fontsize=14, color='gray', ha='center', va='top', alpha=0.8)

    v6graph = io.BytesIO()
    plt.savefig(v6graph, format='png')
    plt.close()

    # Need to seek to zero then return the images in memory.
    v4graph.seek(0)
    v6graph.seek(0)
    return v4graph, v6graph

def createPieGraph(
    entries: pb.pie_subnets_response,
    ) -> Tuple[io.BytesIO, io.BytesIO]:

    logging.info('running createPieGraph')

    # Extract the values and all the smaller prefix lengths
    v4_subnets = []
    v6_subnets = []

    v4_subnets.append(entries.masks.v4_19 + entries.masks.v4_20 + entries.masks.v4_21)
    v4_subnets.append(entries.masks.v4_16 + entries.masks.v4_17 + entries.masks.v4_18)
    v4_subnets.append(entries.masks.v4_22)
    v4_subnets.append(entries.masks.v4_23)
    v4_subnets.append(entries.masks.v4_24)
    v4_labels = ['/19-/21', '/16-/18', '/22', '/23', '/24']
    v4_explode = (0, 0, 0, 0, 0.1)
    v4_colours = ['burlywood', 'lightgreen', 'lightskyblue', 'lightcoral', 'gold']

    v6_subnets.append(entries.masks.v6_32)
    v6_subnets.append(entries.masks.v6_44)
    v6_subnets.append(entries.masks.v6_40)
    v6_subnets.append(entries.masks.v6_36)
    v6_subnets.append(entries.masks.v6_29)
    v6_subnets.append(entries.v6_total - entries.masks.v6_32 -
                entries.masks.v6_44 - entries.masks.v6_40 -
                entries.masks.v6_36 - entries.masks.v6_29 -
                entries.masks.v6_48)
    v6_subnets.append(entries.masks.v6_48)
    v6_labels = ['/32', '/44', '/40', '/36', '/29', 'The Rest', '/48']
    v6_explode = (0, 0, 0, 0, 0, 0, 0.1)
    v6_colours = ['lightgreen', 'burlywood', 'lightskyblue', 'violet', 'linen', 'lightcoral', 'gold']

    # Start with the IPv4 pie
    plt.figure(figsize=(12, 10))
    plt.subplots_adjust(top=1, bottom=0, left=0, right=1, wspace=0)
    plt.suptitle('Current prefix range distribution for IPv4 (' + today + ')', fontsize = 17)
    plt.pie(v4_subnets, labels=v4_labels, colors=v4_colours, explode=v4_explode,
            autopct='%1.1f%%', shadow=True, startangle=90, labeldistance=1.05)
    plt.figtext(0.5, 0.93, copyright,
                fontsize=14, color='gray', ha='center', va='top', alpha=0.8)
    v4pie = io.BytesIO()
    plt.savefig(v4pie, format='png')
    plt.close()

    # Now the IPv6 pie
    plt.figure(figsize=(12, 10))
    plt.subplots_adjust(top=1, bottom=0, left=0, right=1, wspace=0)
    plt.suptitle('Current prefix range distribution for IPv6 (' + today + ')', fontsize = 17)
    plt.pie(v6_subnets, labels=v6_labels, colors=v6_colours, explode=v6_explode,
            autopct='%1.1f%%', shadow=True, startangle=90, labeldistance=1.05)
    plt.figtext(0.5, 0.93, copyright,
                fontsize=14, color='gray', ha='center', va='top', alpha=0.8)
    v6pie = io.BytesIO()
    plt.savefig(v6pie, format='png')
    plt.close()

    # Need to seek to zero then return the images in memory.
    v4pie.seek(0)
    v6pie.seek(0)
    return v4pie, v6pie

def createRPKIPie(
    entries: pb.roas,
    ) -> Tuple[io.BytesIO, io.BytesIO]:
    logging.info('running createRPKIPie')

    # Extract the values and all the smaller prefix lengths
    v4_rpki = []
    v6_rpki = []

    v4_rpki.append(entries.v4_valid)
    v4_rpki.append(entries.v4_invalid)
    v4_rpki.append(entries.v4_unknown)
    v6_rpki.append(entries.v6_valid)
    v6_rpki.append(entries.v6_invalid)
    v6_rpki.append(entries.v6_unknown)

    labels = ['VALID', 'INVALID', 'UNKNOWN']
    colours = ['lightskyblue', 'lightcoral', 'gold']


    # Start with the IPv4 pie
    plt.figure(figsize=(12, 10))
    plt.subplots_adjust(top=1, bottom=0, left=0, right=1, wspace=0)
    plt.suptitle('Current RPKI status for IPv4 (' + today + ')', fontsize = 17)
    plt.pie(v4_rpki, labels=labels, colors=colours,
            autopct='%1.1f%%', shadow=True, startangle=90, labeldistance=1.05)
    plt.figtext(0.5, 0.93, copyright,
                fontsize=14, color='gray', ha='center', va='top', alpha=0.8)
    v4pie = io.BytesIO()
    plt.savefig(v4pie, format='png')
    plt.close()

    # Now the IPv6 pie
    plt.figure(figsize=(12, 10))
    plt.subplots_adjust(top=1, bottom=0, left=0, right=1, wspace=0)
    plt.suptitle('Current RPKI status for IPv6 (' + today + ')', fontsize = 17)
    plt.pie(v6_rpki, labels=labels, colors=colours,
            autopct='%1.1f%%', shadow=True, startangle=90, labeldistance=1.05)
    plt.figtext(0.5, 0.93, copyright,
                fontsize=14, color='gray', ha='center', va='top', alpha=0.8)
    v6pie = io.BytesIO()
    plt.savefig(v6pie, format='png')
    plt.close()

    # Need to seek to zero then return the images in memory.
    v4pie.seek(0)
    v6pie.seek(0)
    return v4pie, v6pie


def createMessage(deltaH: str, deltaW: str) -> str:
    """Creates update message.
    Uses the deltas to formualte a message to be tweeted. Message
    depends on current values, six hour old values, and last weeks values
    """
    logging.info('running createMessage')
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

def tweet(
    account: int,
    message: str,
    image: io.BytesIO,
    ):
    """Tweets to the world.
    Tweets the message, and image if it exists.
    Account used to determine which account to use.
    """
    logging.info('running tweet')

    if account == 4:
        section = 'bgp4_account'
    elif account == 6:
        section = 'bgp6_account'
    
    key = config.get(section, 'consumer_key')
    secret_key = config.get(section, 'consumer_secret')
    token = config.get(section, 'access_token')
    secret_token = config.get(section, 'access_token_secret')

    twitter = Twython(key, secret_key, token, secret_token)
    c = twitter.verify_credentials()
    logging.info("{} account with {} followers verified.".format(c['name'], c['followers_count']))

    # If running a test run, handy to see exactly what will be tweeted.
    if args.dryrun == "True":
        print("account: {}, message: {}".format(account, message))
        if image:
            name = message + ".png"
            with open(name, "wb") as f:
                f.write(image.read())
        return
    # This is where we really send tweets out
    else:
        if image:
            logging.info("Sending the following tweet with an image {}".format(message))
            response = twitter.upload_media(media=image)
            twitter.update_status(status=message, media_ids=[response['media_id']])
        else:
            logging.info("Sending the following tweet without an image {}".format(message))
            twitter.update_status(status=message)


    



if __name__ == "__main__":
    if args.dryrun == "True":
        getCurrent()
        getPrefixPie()
        getRPKIPie()
        getMovement(pb.movement_request.WEEK)
        getMovement(pb.movement_request.MONTH)
        getMovement(pb.movement_request.SIXMONTH)
        getMovement(pb.movement_request.ANNUAL)
        sys.exit("Test finished")

    if args.call == "getcurrent":
        getCurrent()

    elif args.call == "getpie":
        getPrefixPie()
    
    elif args.call == "getrpki":
        getRPKIPie()

    elif args.call == "getmovement":
        if args.time == "week":
            getMovement(pb.movement_request.WEEK)
        elif args.time == "month":
            getMovement(pb.movement_request.MONTH)
        elif args.time == "month6":
            getMovement(pb.movement_request.SIXMONTH)
        elif args.time == "year":
            getMovement(pb.movement_request.ANNUAL)
        else:
            sys.exit("getmovement requires supported time to be set")
    else:
        sys.exit("No option chosen. Use --help to view")
