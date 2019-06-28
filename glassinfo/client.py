#!/usr/bin/env python3

import glass_pb2 as pb
import glass_pb2_grpc
import grpc
import sys


# Set up GRPC server details
grpcserver = "127.0.0.1:7181"
channel = grpc.insecure_channel(grpcserver)
stub = glass_pb2_grpc.looking_glassStub(channel)

# Get info
address = pb.ip_address(
    address = sys.argv[1],
)
a = pb.origin_request(
    ip_address = address,
)

try:
    response = stub.origin(a)
except grpc.RpcError as e:
    #print("Error: {}".format(e))
    print("Error: {}".format(e.details()))
    sys.exit(1)

print("The origin of {} is {}".format(sys.argv[1], response.origin_asn))