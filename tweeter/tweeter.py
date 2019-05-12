#!/usr/bin/env python3

import argparse
import bgpinfo_pb2 as pb
import bgpinfo_pb2_grpc
import configparser
import grpc
import logging
import matplotlib
matplotlib.use('Agg')
import os

# Load config
config = configparser.ConfigParser()
path = "{}/config.ini".format(os.path.dirname(os.path.realpath(__file__)))
config.read(path)
server = str(config.get('grpc', 'server'))
port = str(config.get('grpc', 'port'))
log = config.get('grpc', 'logfile')

# Check arguments
parser = argparse.ArgumentParser()
parser.add_argument("-test", help="test mode will read, but not tweet", default=True)
args = parser.parse_args()

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

    requires:
     - current count
     - count from 6 hours ago
     - count from a week ago
     - How many /24 or /48
    """
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
    delta4 += create_message(ipv4_deltaH, ipv4_deltaW)
    delta6 += create_message(ipv6_deltaH, ipv6_deltaW)
    delta4 += '. ' + str(percent_v4) + '% of prefixes are /24.'
    delta6 += '. ' + str(percent_v6) + '% of prefixes are /48.'

    tweet(4, delta4, None)
    tweet(6, delta6, None)
    setTweetBit(result.time)


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
    requires:
     - current spread of all subnet sizes.
    """
    result = stub.get_pie_subnets(pb.empty())
    print(result)

def setTweetBit(time: str):
    """Set tweet bit.
    Updates database to show the latest tweeted
    values. Useful when comparing historically.
    """
    if args.test:
        print("Will set tweet bit with time {}".format(time))
        return

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

def create_message(deltaH: str, deltaW: str) -> str:
  """Creates update message.
  Uses the deltas to formualte a message to be tweeted. Message
  depends on current values, six hour old values, and last weeks values
  """
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
    account: str,
    message: str,
    image: bytes(),
    ):
    """Tweets to the world.
    Tweets the message, and image if it exists.
    Account used to determine which account to use.
    """
    if args.test:
        print("account: {}, message: {}".format(account, message))
        return



if __name__ == "__main__":
  getCurrent()
  getPrefixPie()