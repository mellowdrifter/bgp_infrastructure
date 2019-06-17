#!/usr/bin/env python3

import grapher
import grapher_pb2 as pb
import unittest

class TestGrapherServicer(unittest.TestCase):

    def test_GetLineGraph(self):
        metadatas = []
        totals_time = []
        metadatas.append(pb.Metadata(
            title = "ipv4 test title",
            x_axis = 12,
            y_axis = 10,
            colours = ["lightgreen", "gold"],
        ))
        metadatas.append(pb.Metadata(
            title = "ipv6 test title",
            x_axis = 12,
            y_axis = 10,
            colours = ["lightgreen", "gold"],
        ))
        totals_time.append(pb.TotalTime(
            v4_values = 10, 
            v6_values = 20,
            time = 1560640600,
            )
        )
        totals_time.append(pb.TotalTime(
            v4_values = 30, 
            v6_values = 40,
            time = 1560640799,
            )
        )
        request = pb.LineGraphRequest(
            metadatas = metadatas,
            totals_time = totals_time,
            copyright = "copyright",
        )

        result = grapher.get_line_graph(request)
        print(type(result))
        print(result)
        print(result.images[0])




    #def test_GetPieChart(self):


if __name__ == '__main__':
    unittest.main()