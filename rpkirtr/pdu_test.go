package main

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSerialNotifyPDU(t *testing.T) {
	type serialPDU struct {
		Version uint8
		Ptype   uint8
		Session uint16
		Length  uint32
		Serial  uint32
	}
	pdus := []struct {
		desc    string
		session uint16
		serial  uint32
	}{
		{
			desc:    "regular values",
			session: 123,
			serial:  456,
		},
		{
			desc: "zero values",
		},
	}

	for _, p := range pdus {

		// Send data to be encoded
		var buffer bytes.Buffer
		pdu := &serialNotifyPDU{
			Session: p.session,
			Serial:  p.serial,
		}
		pdu.serialize(&buffer)

		// Read data back that was written
		buf := bytes.NewReader(buffer.Bytes())
		var got serialPDU
		binary.Read(buf, binary.BigEndian, &got)

		// Directly create PDU
		want := serialPDU{
			Version: version1,
			Ptype:   serialNotify,
			Session: p.session,
			Length:  12,
			Serial:  p.serial,
		}

		// Compare them
		if !cmp.Equal(got, want) {
			t.Errorf("PDU encoded is not what was expected. Got %+v, Wanted %+v\n", got, want)
		}
	}
}
