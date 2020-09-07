#!/usr/bin/env python3

from typing import List
from typing import Text
import os
from concurrent import futures
import configparser
import grapher_pb2 as pb
import grapher_pb2_grpc
import datetime
import grpc
import io
import logging
import matplotlib
matplotlib.use('Agg')

_PORT = os.environ["PORT"]


class Grapher(grapher_pb2_grpc.GrapherServicer):
    """Provides methods that implement functionality of the grapher server."""

    def GetLineGraph(self, request, context):
        logging.info("Request received for GetLineGraph")
        return get_line_graph(request)

    def GetPieChart(self, request, context):
        logging.info("Request received for GetPieChart")
        return get_pie_chart(request)

    def GetRPKI(self, request, context):
        logging.info("Request received for GetRPKI")
        return get_rpki(request)

    def TestRPC(self, request, context):
        logging.info("Received request: %s", request)
        return pb.TestResponse(testresponse="something")


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
        dates.append(datetime.datetime.fromtimestamp(
            request.totals_time[i].time))
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
        theme = metadata.theme

        image = io.BytesIO()
        matplotlib.pyplot.figure(figsize=(x, y))
        ax = matplotlib.pyplot.subplot(111)
        xfmt = matplotlib.dates.DateFormatter('%Y-%m-%d')
        ax.xaxis.set_major_formatter(xfmt)
        matplotlib.pyplot.suptitle(title, fontsize=17)
        ax.grid(True)
        ax.spines["top"].set_visible(False)
        ax.spines["bottom"].set_visible(False)
        ax.spines["right"].set_visible(False)
        ax.spines["left"].set_visible(False)
        ax.get_xaxis().tick_bottom()
        ax.get_yaxis().tick_left()
        matplotlib.pyplot.xticks(fontsize=12, rotation=12)
        matplotlib.pyplot.yticks(fontsize=12)
        matplotlib.pyplot.ticklabel_format(
            axis='y', style='plain', useOffset=False)
        matplotlib.pyplot.tick_params(axis="both", which="both", bottom=False, top=False,
                                      labelbottom=True, left=False, right=False, labelleft=True)
        matplotlib.pyplot.plot(
            dates, totals[j], 'o-', lw=1, alpha=0.4, color=colour)
        if theme == "dark":
            matplotlib.pyplot.figtext(0.5, 0.93, request.copyright,
                                      fontsize=14, color='snow', ha='center', va='top', alpha=0.8)
        else:
            matplotlib.pyplot.figtext(0.5, 0.93, request.copyright,
                                      fontsize=14, color='gray', ha='center', va='top', alpha=0.8)

        matplotlib.pyplot.savefig(image, format='png')
        image.seek(0)
        graph = graphs.images.add()
        graph.image = image.read()
        graph.title = title
        matplotlib.pyplot.close()
        j += 1

    logging.info("Returning line graphs")
    return graphs


def get_pie_chart(
    request: pb.PieChartRequest()
) -> pb.GrapherResponse():

    logging.info('running get_line_graph')

    pieCharts = pb.GrapherResponse()
    subnets = [list(request.subnets.v4_values),
               list(request.subnets.v6_values)]

    j = 0
    for metadata in request.metadatas:
        title = metadata.title
        x = metadata.x_axis
        y = metadata.y_axis
        labels = list(metadata.labels)
        colours = list(metadata.colours)
        theme = metadata.theme

        explode = [float(0)] * (len(colours) - 1)
        explode.append(0.1)

        image = io.BytesIO()
        matplotlib.pyplot.figure(figsize=(x, y))
        matplotlib.pyplot.subplots_adjust(
            top=1, bottom=0, left=0, right=1, wspace=0)
        matplotlib.pyplot.suptitle(title, fontsize=17)
        matplotlib.pyplot.pie(subnets[j], labels=labels, colors=colours, explode=explode,
                              autopct='%1.1f%%', shadow=True, startangle=90, labeldistance=1.05)
        if theme == "dark":
            matplotlib.pyplot.figtext(0.5, 0.93, request.copyright,
                                      fontsize=14, color='snow', ha='center', va='top', alpha=0.8)
        else:
            matplotlib.pyplot.figtext(0.5, 0.93, request.copyright,
                                      fontsize=14, color='gray', ha='center', va='top', alpha=0.8)
        matplotlib.pyplot.savefig(image, format='png')
        image.seek(0)
        pie = pieCharts.images.add()
        pie.image = image.read()
        pie.title = title
        matplotlib.pyplot.close()
        j += 1

    logging.info("Returning pie charts")
    return pieCharts

# TODO:
# Show the actual amounts on the graph.
# UNKNOWN should also show None
# How many source ASs?


def get_rpki(
    request: pb.RPKIRequest()
) -> pb.GrapherResponse():

    logging.info('running get_rpki')

    v4_rpki = []
    v6_rpki = []

    v4_rpki.append(request.rpkis.v4_valid)
    v4_rpki.append(request.rpkis.v4_invalid)
    v4_rpki.append(request.rpkis.v4_unknown)
    v6_rpki.append(request.rpkis.v6_valid)
    v6_rpki.append(request.rpkis.v6_invalid)
    v6_rpki.append(request.rpkis.v6_unknown)
    rpkis = [v4_rpki, v6_rpki]
    RPKICharts = pb.GrapherResponse()

    labels = ['VALID', 'INVALID', 'NO ROA (UNKNOWN)']
    colours = ['lightskyblue', 'lightcoral', 'gold']

    j = 0
    for metadata in request.metadatas:
        title = metadata.title
        x = metadata.x_axis
        y = metadata.y_axis
        theme = metadata.theme

        # Start with something
        image = io.BytesIO()
        matplotlib.pyplot.figure(figsize=(x, y))
        matplotlib.pyplot.subplots_adjust(
            top=1, bottom=0, left=0, right=1, wspace=0)
        matplotlib.pyplot.suptitle(title, fontsize=17)
        matplotlib.pyplot.pie(rpkis[j], labels=labels, colors=colours,
                              autopct='%1.1f%%', shadow=True, startangle=90, labeldistance=1.05)
        if theme == "dark":
            matplotlib.pyplot.figtext(0.5, 0.93, request.copyright,
                                      fontsize=14, color='snow', ha='center', va='top', alpha=0.8)
        else:
            matplotlib.pyplot.figtext(0.5, 0.93, request.copyright,
                                      fontsize=14, color='gray', ha='center', va='top', alpha=0.8)

        matplotlib.pyplot.savefig(image, format='png')
        image.seek(0)
        rpki = RPKICharts.images.add()
        rpki.image = image.read()
        rpki.title = title
        matplotlib.pyplot.close()
        j += 1

    logging.info("Returning rpki charts")
    return RPKICharts

# Text


def _serve(port: Text):
    bind_address = f"[::]:{port}"
    server = grpc.server(futures.ThreadPoolExecutor())
    grapher_pb2_grpc.add_GrapherServicer_to_server(Grapher(), server)
    server.add_insecure_port(bind_address)
    server.start()
    logging.info("Listening on %s.", bind_address)
    server.wait_for_termination()


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    _serve(_PORT)
