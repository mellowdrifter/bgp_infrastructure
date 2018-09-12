#!/usr/bin/env python3

import bgpinfo_pb2 as pb
import bgpinfo_pb2_grpc
import configparser
import grpc
import birdparse
import time


# Load config
config = configparser.ConfigParser()
config.read("config.ini")
server = str(config.get('grpc', 'server'))
port = str(config.get('grpc', 'port'))

#Set up GRPC server details
grpcserver = "%s:%s" % (server, port)
channel = grpc.insecure_channel(grpcserver)
stub = bgpinfo_pb2_grpc.bgp_infoStub(channel)

def main():

    # Prefix counts.
    bgp4, bgp6 = birdparse.getTotals()
    prefix_count = []
    prefix4 = pb.prefix_count(
        family = pb.address_family.Value('IPV4'),
        active = int(bgp4[1]),
        total = int(bgp4[0]),
    )
    prefix6 = pb.prefix_count(
        family = pb.address_family.Value('IPV6'),
        active = int(bgp6[1]),
        total = int(bgp6[0]),
    )

    prefix_count.append(prefix4)
    prefix_count.append(prefix6)


    # Peer Count.
    peers4, state4 = birdparse.getPeers(4)
    peers6, state6 = birdparse.getPeers(6)

    peers = []
    peers4 = pb.peer_count(
        family = pb.address_family.Value('IPV4'),
        configured = peers4,
        up = state4
    )
    peers6 = pb.peer_count(
        family = pb.address_family.Value('IPV6'),
        configured = peers6,
        up = state6
    )
    peers.append(peers4)
    peers.append(peers6)


    # AS number count.
    as4, as6, as10, as4_only, as6_only, as_both = birdparse.getSrcAS()
    as_count = pb.as_count(
        as4 = as4,
        as6 = as6,
        as10 = as10,
        as4_only = as4_only,
        as6_only = as6_only,
        as_both = as_both,
    )


    # Memory use.
    bgp4Mem = birdparse.getMem(4)
    bgp6Mem = birdparse.getMem(6)

    memory = []
    mem4 = pb.memory(
        family = pb.address_family.Value('IPV4'),
        tables = bgp4Mem['Routing tables'],
        total = bgp4Mem['Total'],
        protocols = bgp4Mem['Protocols'],
        attributes = bgp4Mem['Route attributes'],
        roa = bgp4Mem['ROA tables'],
    )
    mem6 = pb.memory(
        family = pb.address_family.Value('IPV6'),
        tables = bgp6Mem['Routing tables'],
        total = bgp6Mem['Total'],
        protocols = bgp6Mem['Protocols'],
        attributes = bgp6Mem['Route attributes'],
        roa = bgp6Mem['ROA tables'],
    )
    memory.append(mem4)
    memory.append(mem6)


    bgp4Subnets = masker(4, birdparse.getSubnets(4))
    bgp6Subnets = masker(6, birdparse.getSubnets(6))

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
        large_community = large,
        masks = bgp4Subnets + bgp6Subnets,
    )

    return current_values

def masker(family, masks):
    non_zero = []
    if family == 4:
        family = pb.address_family.Value('IPV4')
    else:
        family = pb.address_family.Value('IPV6')
    for k, v in masks.items():
        if v != 0:
            non_zero.append(pb.mask(
                address_family = family,
                mask = int(k),
                active = int(v),
            ))
    return non_zero



if __name__ == "__main__":
    current_values = main()
    result = stub.add_latest(current_values)
    print(result)
