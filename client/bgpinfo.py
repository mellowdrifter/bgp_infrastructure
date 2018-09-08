#!/usr/bin/env python3

import bgpinfo_pb2 as pb
import bgpinfo_pb2_grpc
import configparser
import grpc
import time
import info


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

    # Grab prefixes from BIRD
    bgp4 = info.getTotals(4)
    bgp6 = info.getTotals(6)

    # as path numbers
    as4_len, as6_len, as10_len, as4_only, as6_only, as_both = info.getSrcAS()

    data = []
    bgp4Subnets = info.getSubnets(4)
    bgp6Subnets = info.getSubnets(6)
    bgp4Mem = info.getMem(4)
    bgp6Mem = info.getMem(6)
    data.append(bgp4[0])
    data.append(bgp4[1])
    data.append(bgp6[0])
    data.append(bgp6[1])
    peers4 = info.getPeers(4)
    peers6 = info.getPeers(6)

    current_values = pb.values(
        time = int(time.time()),
    )

    return current_values


if __name__ == "__main__":
    current_values = main()
    result = stub.add_latest(current_values)
    print(result)
