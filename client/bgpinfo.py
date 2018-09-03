import bgpinfo_pb2 as pb
import bgpinfo_pb2_grpc
import configparser
import grpc
import time


# Load config
config = configparser.ConfigParser()
config.read("config.ini")
server = str(config.get('sensor', 'server'))
port = str(config.get('sensor', 'port'))

#Set up GRPC server details
grpcserver = "%s:%s" % (server, port)
channel = grpc.insecure_channel(grpcserver)
stub = bgpinfo_pb2_grpc.sensor_dataStub(channel)


if __name__ == "__main__":
    current_values = pb.values(
        time = int(time.time()),
    )
    result = stub.add_latest(current_values)
    print(result)