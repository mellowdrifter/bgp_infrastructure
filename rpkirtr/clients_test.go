package main

import (
	"net"
	"testing"
	"time"
)

func TestWritePrefixPDU(t *testing.T) {
	tests := []struct {
		desc string
		roa  roa
		flag uint8
	}{
		{
			desc: "test 1",
			roa: roa{
				Prefix:  "192.168.0.0",
				MinMask: 16,
				MaxMask: 24,
				ASN:     9876554,
				RIR:     3,
				IsV4:    true,
			},
			flag: announce,
		},
	}
	for _, v := range tests {
		ln, _ := net.Listen("tcp", ":1123")
		go func() {
			defer ln.Close()
			ln.Accept()
		}()

		conn, _ := net.Dial("tcp", ln.Addr().String())
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(time.Second * 5))

		writePrefixPDU(&v.roa, conn, v.flag)

		out := make([]byte, 1024)
		if _, err := conn.Read(out); err != nil {
			t.Errorf("%#v\n", out)
		}
		//buf, _ := ioutil.ReadAll(conn)
	}
}
