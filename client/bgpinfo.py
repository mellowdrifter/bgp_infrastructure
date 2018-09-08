#!/usr/bin/env python3

import bgpinfo_pb2 as pb
import bgpinfo_pb2_grpc
import configparser
import grpc
import parser
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
    bgp4 = parser.getTotals(4)
    bgp6 = parser.getTotals(6)
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
    peer4count = parser.getPeers(4)
    peer6count = parser.getPeers(6)

    peers = []
    peers4 = pb.peer_count(
        family = pb.address_family.Value('IPV4'),
        configured = int(peer4count['peersConfigured']),
        up = int(peer4count['peersUp']),
    )
    peers6 = pb.peer_count(
        family = pb.address_family.Value('IPV6'),
        configured = int(peer6count['peersConfigured']),
        up = int(peer6count['peersUp']),
    )
    peers.append(peers4)
    peers.append(peers6)


    # AS number count.
    as4, as6, as10, as4_only, as6_only, as_both = parser.getSrcAS()
    as_count = pb.as_count(
        as4 = as4,
        as6 = as6,
        as10 = as10,
        as4_only = as4_only,
        as6_only = as6_only,
        as_both = as_both,
    )


    # Memory use.
    bgp4Mem = parser.getMem(4)
    bgp6Mem = parser.getMem(6)

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


    bgp4Subnets = parser.getSubnets(4)
    bgp6Subnets = parser.getSubnets(6)

    current_values = pb.values(
        time = int(time.time()),
        prefix_count = prefix_count,
        as_count = as_count,
        peers = peers,
        mem_use = memory,
    )

    return current_values


if __name__ == "__main__":
    current_values = main()
    result = stub.add_latest(current_values)
    print(result)
