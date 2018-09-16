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
    prefix_count = pb.prefix_count(
        active_4 = int(bgp4[1]),
        total_4 = int(bgp4[0]),
        active_6 = int(bgp6[1]),
        total_6 = int(bgp6[0]),
    )

    # Peer Count.
    peers4, state4 = birdparse.getPeers(4)
    peers6, state6 = birdparse.getPeers(6)

    peers = pb.peer_count(
        peer_count_4 = peers4,
        peer_up_4 = state4,
        peer_count_6 = peers6,
        peer_up_6 = state6,

    )

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

    mask4, mask6 = birdparse.getSubnets()
    masks = masker(mask4, mask6)

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
    )
    return masks





if __name__ == "__main__":
    current_values = main()
    print("Finished gathering data. Now sending to server")
    result = stub.add_latest(current_values)
    print(result)
