package main

import (
	"io/ioutil"
	"reflect"
	"testing"

	pb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpinfo"
)

var source = []byte(`
<html><head><title>AS Names</title>
<meta http-equiv="Content-Type" content="text/html; charset=ISO-8859-1">
<style>
<!--
A:link {text-decoration: none}
A:visited {text-decoration: none}
A:active {text-decoration: none}
-->
</style>
</head>
<body>
<br>
<HR>
<PRE>
<a href="/cgi-bin/as-report?as=AS0&view=2.0">AS0    </a> -Reserved AS-, ZZ
<a href="/cgi-bin/as-report?as=AS1&view=2.0">AS1    </a> LVLT-1 - Level 3 Parent, LLC, US
<a href="/cgi-bin/as-report?as=AS2&view=2.0">AS2    </a> UDEL-DCN - University of Delaware, US
<a href="/cgi-bin/as-report?as=AS3&view=2.0">AS3    </a> MIT-GATEWAYS - Massachusetts Institute of Technology, US
<a href="/cgi-bin/as-report?as=AS4&view=2.0">AS4    </a> ISI-AS - University of Southern California, US
<a href="/cgi-bin/as-report?as=AS5&view=2.0">AS5    </a> SYMBOLICS - Symbolics, Inc., US
</PRE>
<HR>
<I>File last modified at Sat Jul 20 13:16:22 2019
 (UTC+1000)</I>
</body>
</html>
`)

var good = []*pb.AsnName{
	&pb.AsnName{
		AsName:   "-Reserved AS-",
		AsLocale: "ZZ",
	},
	&pb.AsnName{
		AsName:   "LVLT-1 - Level 3 Parent, LLC",
		AsNumber: 1,
		AsLocale: "US",
	},
	&pb.AsnName{
		AsName:   "UDEL-DCN - University of Delaware",
		AsNumber: 2,
		AsLocale: "US",
	},
	&pb.AsnName{
		AsName:   "MIT-GATEWAYS - Massachusetts Institute of Technology",
		AsNumber: 3,
		AsLocale: "US",
	},
	&pb.AsnName{
		AsName:   "ISI-AS - University of Southern California",
		AsNumber: 4,
		AsLocale: "US",
	},
	&pb.AsnName{
		AsName:   "SYMBOLICS - Symbolics, Inc.",
		AsNumber: 5,
		AsLocale: "US",
	},
}

func TestDecoder(t *testing.T) {
	output := decoder(source)
	if !reflect.DeepEqual(output, good) {
		t.Errorf("%+v\n%+v\n", output, good)
	}

}

func TestDecoderFull(t *testing.T) {
	data, err := ioutil.ReadFile("autnums.html")
	if err != nil {
		panic(err)
	}
	output := decoder(data)
	if len(output) != 92093 {
		t.Errorf("Amount of ASs should be 92093, but got %d", len(output))
	}
	for _, info := range output {
		if info.GetAsLocale() == "" {
			t.Errorf("AS %s has no Locale", info.GetAsName())
		}
	}

}
