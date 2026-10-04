#!/usr/bin/env python3

import app as grapher
import grapher_pb2 as pb
import unittest

PNG_HEADER = b'\x89PNG\r\n\x1a\n'

class TestGrapherServicer(unittest.TestCase):

    def test_get_line_graph(self):

        totals_time = []
        metadatas = []

        metadatas.append(pb.Metadata(
            title = "IPv4 title",
            x_axis = 12,
            y_axis = 10,
            colour = "#238341",
            )
        )
        metadatas.append(pb.Metadata(
            title = "IPv6 title",
            x_axis = 12,
            y_axis = 10,
            colour = "#0041A0",
            )
        )
        totals_time.append(pb.TotalTime(
            v4_values = 10, 
            v6_values = 20,
            time = 1560640600,
            )
        )
        totals_time.append(pb.TotalTime(
            v4_values = 30, 
            v6_values = 40,
            time = 1560740799,
            )
        )
        totals_time.append(pb.TotalTime(
            v4_values = 25, 
            v6_values = 35,
            time = 1560840998,
            )
        )
        request = pb.LineGraphRequest(
            metadatas = metadatas,
            totals_time = totals_time,
            copyright = "some copyright",
        )

        results = grapher.get_line_graph(request).images
        self.assertEqual(len(results), 2)
        for i in range(len(results)):
            self.assertTrue(results[i].image.startswith(PNG_HEADER))
            self.assertGreater(len(results[i].image), 1000)


    def test_get_pie_chart(self):

        metadatas = []

        metadatas.append(pb.Metadata(
            title = "IPv4 title",
            x_axis = 12,
            y_axis = 10,
            colours = ["lightgreen", "gold"],
            labels = ["/8", "/24"],
            )
        )
        metadatas.append(pb.Metadata(
            title = "IPv6 title",
            x_axis = 12,
            y_axis = 10,
            colours = ["lightgreen", "gold"],
            labels = ["/8", "/24"],
            )
        )
        subnet_family = pb.SubnetFamily(
            v4_values = (300, 600),
            v6_values = (30, 100),
        )

        request = pb.PieChartRequest(
            metadatas = metadatas,
            subnets = subnet_family,
            copyright = "some copyright",
        )

        results = grapher.get_pie_chart(request).images
        self.assertEqual(len(results), 2)
        for i in range(len(results)):
            self.assertTrue(results[i].image.startswith(PNG_HEADER))
            self.assertGreater(len(results[i].image), 1000)


    def test_get_rpki(self):

        rpkis = []
        metadatas = []

        metadatas.append(pb.Metadata(
            title = "IPv4 title",
            x_axis = 12,
            y_axis = 10,
            )
        )
        metadatas.append(pb.Metadata(
            title = "IPv6 title",
            x_axis = 12,
            y_axis = 10,
            )
        )
        rpki = pb.RPKI(
            v4_valid = 100,
            v4_invalid = 100,
            v4_unknown = 100,
            v6_valid = 100,
            v6_invalid = 100,
            v6_unknown = 100,
        )

        request = pb.RPKIRequest(
            metadatas = metadatas,
            rpkis = rpki,
            copyright = "some copyright",
        )

        results = grapher.get_rpki(request).images
        self.assertEqual(len(results), 2)
        for i in range(len(results)):
            self.assertTrue(results[i].image.startswith(PNG_HEADER))
            self.assertGreater(len(results[i].image), 1000)


if __name__ == '__main__':
    unittest.main()