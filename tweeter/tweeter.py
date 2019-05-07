#!/usr/bin/env python3

import configparser
import grpc
import logging
import os

# Load config
config = configparser.ConfigParser()
path = "{}/config.ini".format(os.path.dirname(os.path.realpath(__file__)))
config.read(path)
server = str(config.get('grpc', 'server'))
port = str(config.get('grpc', 'port'))
log = config.get('grpc', 'logfile')

# Set up logging
format_string = '%(levelname)s: %(asctime)s: %(message)s'
logging.basicConfig(filename=log, level=logging.INFO, format=format_string)

# Set up GRPC server details
grpcserver = "%s:%s" % (server, port)
channel = grpc.insecure_channel(grpcserver)
stub = bgpinfo_pb2_grpc.bgp_infoStub(channel)

def getCurrent():
    """Grabs current v4 and v6 table count.
    This function will grab the current
    v4 and v6 count to tweet.
    """

def getWeek():
    """Grabs weekly data for tweet.
    This function will grab the v4 and v6 
    counts over the last week.
    """

def getMonth():
    """Grabs monthly data for tweet.
    This function will grab the v4 and v6 
    counts over the last month.
    """

def get6Month():
    """Grabs semi-annual data for tweet.
    This function will grab the v4 and v6 
    counts over the last six months.
    """

def getAnnual():
    """Grabs annual data for tweet.
    This function will grab the v4 and v6 
    counts over the last six year.
    """

def getPrefixPie():
    """Create Pie graph.
    Grab the latest subnet size count and
    create pie graphs with those counts
    for each address family
    """

def setTweetBit():
    """Set tweet bit.
    Updates database to show the latest tweeted
    values. Useful when comparing historically.
    """

def plotGraph(
    entries: list(),
    family: int,
    time_period: str,
    ) -> bytes():
    """Creates a plotted graph.
    Uses entries and time_period to create a
    matplotlib-based graphf for the respective
    address family.
    """

def tweet(
    account: str,
    image: bytes(),
    message: str
    ):
    """Tweets to the world.
    Tweets the message, and image if it exists.
    Account used to determine which account to use.
    """



if __name__ == "__main__":
  do_something)()