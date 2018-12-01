#!/usr/bin/env python3

import bgpinfo_pb2 as pb
import bgpinfo_pb2_grpc
import birdparse
import checkBird
import configparser
import grpc
import logging
import os
import time


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

#Set up GRPC server details
grpcserver = "%s:%s" % (server, port)
channel = grpc.insecure_channel(grpcserver)
stub = bgpinfo_pb2_grpc.bgp_infoStub(channel)

def get_data():

    # Prefix counts.
    logging.info('prefix count')
    bgp4, bgp6 = birdparse.getTotals()
    prefix_count = pb.prefix_count(
        active_4 = int(bgp4[1]),
        total_4 = int(bgp4[0]),
        active_6 = int(bgp6[1]),
        total_6 = int(bgp6[0]),
    )

    # Peer Count.
    logging.info('peer count')
    peers4, state4 = birdparse.getPeers(4)
    peers6, state6 = birdparse.getPeers(6)

    peers = pb.peer_count(
        peer_count_4 = peers4,
        peer_up_4 = state4,
        peer_count_6 = peers6,
        peer_up_6 = state6,

    )

    # AS number count.
    logging.info('AS numbers')
    as4, as6, as10, as4_only, as6_only, as_both = birdparse.getSrcAS()
    as_count = pb.as_count(
        as4 = as4,
        as6 = as6,
        as10 = as10,
        as4_only = as4_only,
        as6_only = as6_only,
        as_both = as_both,
    )


    # Memory use
    # TODO: Remove this from everywhere. I really don't care about memory usage
    logging.info('memory')
    bgp4Mem = birdparse.getMem(4)
    bgp6Mem = birdparse.getMem(6)

    memory = []
    mem4 = pb.memory(
        family = pb.address_family.Value('IPV4'),
        memstats = pb.memstats(
            tables = bgp4Mem['Routing tables'],
            total = bgp4Mem['Total'],
            protocols = bgp4Mem['Protocols'],
            attributes = bgp4Mem['Route attributes'],
        )
    )
    mem6 = pb.memory(
        family = pb.address_family.Value('IPV6'),
        memstats = pb.memstats(
            tables = bgp6Mem['Routing tables'],
            total = bgp6Mem['Total'],
            protocols = bgp6Mem['Protocols'],
            attributes = bgp6Mem['Route attributes'],
        )
    )
    memory.append(mem4)
    memory.append(mem6)

    logging.info('subnets')
    mask4, mask6 = birdparse.getSubnets()
    masks = masker(mask4, mask6)

    # TODO: Fix this. Seems to be wrong
    logging.info('large communities')
    large4, large6 = birdparse.getLargeCommunitys()
    large = pb.large_community(
        c4 = large4,
        c6 = large6,
    )

    current_values = pb.values(
        time = int(time.time()),
        prefix_count = prefix_count,
        as_count = as_count,
        peers = peers,
        mem_use = memory,
        masks = masks,
        large_community = large,
    )

    return current_values

def masker(mask4, mask6):
    masks = pb.masks(
        v4_08 = mask4[0],
        v4_09 = mask4[1],
        v4_10 = mask4[2],
        v4_11 = mask4[3],
        v4_12 = mask4[4],
        v4_13 = mask4[5],
        v4_14 = mask4[6],
        v4_15 = mask4[7],
        v4_16 = mask4[8],
        v4_17 = mask4[9],
        v4_18 = mask4[10],
        v4_19 = mask4[11],
        v4_20 = mask4[12],
        v4_21 = mask4[13],
        v4_22 = mask4[14],
        v4_23 = mask4[15],
        v4_24 = mask4[16],
        v6_08 = mask6[0],
        v6_09 = mask6[1],
        v6_10 = mask6[2],
        v6_11 = mask6[3],
        v6_12 = mask6[4],
        v6_13 = mask6[5],
        v6_14 = mask6[6],
        v6_15 = mask6[7],
        v6_16 = mask6[8],
        v6_17 = mask6[9],
        v6_18 = mask6[10],
        v6_19 = mask6[11],
        v6_20 = mask6[12],
        v6_21 = mask6[13],
        v6_22 = mask6[14],
        v6_23 = mask6[15],
        v6_24 = mask6[16],
        v6_25 = mask6[17],
        v6_26 = mask6[18],
        v6_27 = mask6[19],
        v6_28 = mask6[20],
        v6_29 = mask6[21],
        v6_30 = mask6[22],
        v6_31 = mask6[23],
        v6_32 = mask6[24],
        v6_33 = mask6[25],
        v6_34 = mask6[26],
        v6_35 = mask6[27],
        v6_36 = mask6[28],
        v6_37 = mask6[29],
        v6_38 = mask6[30],
        v6_39 = mask6[31],
        v6_40 = mask6[32],
        v6_41 = mask6[33],
        v6_42 = mask6[34],
        v6_43 = mask6[35],
        v6_44 = mask6[36],
        v6_45 = mask6[37],
        v6_46 = mask6[38],
        v6_47 = mask6[39],
        v6_48 = mask6[40],
    )
    return masks

if __name__ == "__main__":
    if not checkBird.isRun('bird6') and not checkBird.isRun('bird'):
        logging.info('both bird and bird6 are not running')
        exit()
    if not checkBird.isRun('bird'):
        logging.info('bird is not running')
        exit()
    if not checkBird.isRun('bird6'):
        logging.info('bird6 is not running')
        exit()

    logging.info('Gathering data')
    current_values = get_data()

    logging.info('Sending data to server')
    result = stub.add_latest(current_values)
    logging.info('server response: ' + str(result))
