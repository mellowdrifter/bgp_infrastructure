#!/usr/bin/env python3

import configparser
import grapher_pb2 as pb
import grapher_pb2_grpc
import datetime
import grpc
import io
import logging
import matplotlib
matplotlib.use('Agg')
from concurrent import futures
from matplotlib import pyplot as plt
from matplotlib import dates as mdates
import os
import sys
import time
from typing import List


_ONE_DAY_IN_SECONDS = 60 * 60 * 24

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


class GrapherServicer(grapher_pb2_grpc.GrapherServicer):
    """Provides methods that implement functionality of the grapher server."""

    def GetLineGraph(self, request, context):
        return get_line_graph(request)


    def GetPieChart(self, request, context):
        return get_pie_chart(request)


def get_line_graph(
    request: pb.LineGraphRequest()
    ) -> pb.GrapherResponse():

    logging.info('running get_line_graph')

    totals = []
    dates = []
    v4totals = []
    v6totals = []
    graphs = pb.GrapherResponse()

    for i in range(len(request.totals_time)):
        dates.append(datetime.datetime.fromtimestamp(request.totals_time[i].time))
        v4totals.append(request.totals_time[i].v4_values)
        v6totals.append(request.totals_time[i].v6_values)
    totals.append(v4totals)
    totals.append(v6totals)

    j = 0
    for metadata in request.metadatas:
        title = metadata.title
        x = metadata.x_axis
        y = metadata.y_axis
        colour = metadata.colour


        #print(title, x, y, labels, colours, explode)
        image = io.BytesIO()
        plt.figure(figsize=(x, y))
        ax = plt.subplot(111)
        xfmt = mdates.DateFormatter('%Y-%m-%d')
        ax.xaxis.set_major_formatter(xfmt)
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
        plt.plot(dates, totals[j], 'o-', lw=1, alpha=0.4, color=colour)
        plt.figtext(0.5, 0.93, request.copyright,
                    fontsize=14, color='gray', ha='center', va='top', alpha=0.8)

        plt.savefig(image, format='png')
        image.seek(0)
        graph = pb.Image(
            image = image.read(),
            title = title,
        )
        graphs.images.append(graph)
        plt.close()
        j+=1

    logging.info("Returning line graphs")
    return graphs


def get_pie_chart(
    request: pb.PieChartRequest()
    ) -> pb.GrapherResponse():

    logging.info('running get_line_graph')

    subnets = []
    pieCharts = pb.GrapherResponse()
    subnets.append(list(request.subnets.v4_values))
    subnets.append(list(request.subnets.v6_values))

    j = 0
    for metadata in request.metadatas:
        title = metadata.title
        x = metadata.x_axis
        y = metadata.y_axis
        labels = list(metadata.labels)
        colours = list(metadata.colours)

        explode = [float(0)] * (len(colours) - 1)
        explode.append(0.1)

        #print(title, x, y, labels, colours, explode)


    
        # Start with something
        image = io.BytesIO()
        plt.figure(figsize=(x, y))
        plt.subplots_adjust(top=1, bottom=0, left=0, right=1, wspace=0)
        plt.suptitle(title, fontsize = 17)
        plt.pie(subnets[j], labels=labels, colors=colours, explode=explode,
                autopct='%1.1f%%', shadow=True, startangle=90, labeldistance=1.05)
        plt.figtext(0.5, 0.93, request.copyright,
                    fontsize=14, color='gray', ha='center', va='top', alpha=0.8)
        plt.savefig(image, format='png')
        image.seek(0)
        pie = pb.Image(
            image = image.read(),
            title = title,
        )
        pieCharts.images.append(pie)
        plt.close()
        j+=1
    
    logging.info("Returning pie charts")
    return pieCharts




if __name__ == "__main__":
    grpcserver = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    grapher_pb2_grpc.add_GrapherServicer_to_server(
        GrapherServicer(), grpcserver
    )

    logging.info('Listening on port {}.'.format(port))
    grpcserver.add_insecure_port("{}:{}".format(server, port))
    grpcserver.start()

    # since grpcserver.start() will not block,
    # a sleep-loop is added to keep alive
    try:
        while True:
            time.sleep(_ONE_DAY_IN_SECONDS)
    except KeyboardInterrupt:
        print('Keyboard interrupted. Stopping server.')
        grpcserver.stop(0)