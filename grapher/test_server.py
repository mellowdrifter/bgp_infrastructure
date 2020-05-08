#!/usr/bin/env python3

import grapher
import grapher_pb2 as pb
import hashlib
import unittest

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
        hashes = [
            "b2c242eb9d89dc5499ff2bbd28743cf3f335ba6100300da4b3d4237e8c685f2f",
            "9a0a6c6c9a9c7b647bae8a687c4afe17c941e0bc18953e8ba8a85334fcf877f8",
            ]

        for i in range(len(results)):
            # Uncomment the below when making changes to save the image to view.
            #image = ("{}.png".format(results[i].title))
            #print("hash of file is {}".format(hashlib.sha256(results[i].image).hexdigest()))
            #with open(image, "wb") as f:
            #    f.write(results[i].image)
            self.assertEqual(
                hashlib.sha256(results[i].image).hexdigest(), 
                hashes[i],
            )


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
        hashes = [
            "eff79e5c555edfebce3be57e0cf70ebda366dadb8d435063f89ff5e5461aa636",
            "7d724137b605f36abe1d44ac088db2dccd182eb205896b3e9177d581e047ca0b",
            ]
        for i in range(len(results)):
            #Uncomment the below when making changes to save the image to view.
            #image = ("{}.png".format(results[i].title))
            #print("hash of file is {}".format(hashlib.sha256(results[i].image).hexdigest()))
            #with open(image, "wb") as f:
            #    f.write(results[i].image)
            self.assertEqual(
                hashlib.sha256(results[i].image).hexdigest(), 
                hashes[i],
            )


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
        hashes = [
            "e258018ac4eca9257405419ee9d8a85a707534857f47394b00fd092b8657be66",
            "d1a15e5116ec95af275b6cface9aceeb5db818916250295ea85e6c5c912e86ac",
            ]
        for i in range(len(results)):
            #Uncomment the below when making changes to save the image to view.
            #image = ("{}.png".format(results[i].title))
            #print("hash of file is {}".format(hashlib.sha256(results[i].image).hexdigest()))
            #with open(image, "wb") as f:
            #    f.write(results[i].image)
            self.assertEqual(
                hashlib.sha256(results[i].image).hexdigest(), 
                hashes[i],
            )


if __name__ == '__main__':
    unittest.main()