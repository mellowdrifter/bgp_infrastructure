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

func TestCacheResponsePDU(t *testing.T) {
	type cachePDU struct {
		Version uint8
		Ptype   uint8
		Session uint16
		Length  uint32
	}
	pdus := []struct {
		desc    string
		session uint16
	}{
		{
			desc:    "regular values",
			session: 123,
		},
		{
			desc: "zero values",
		},
	}

	for _, p := range pdus {

		// Send data to be encoded
		var buffer bytes.Buffer
		pdu := &cacheResponsePDU{
			sessionID: p.session,
		}
		pdu.serialize(&buffer)

		// Read data back that was written
		buf := bytes.NewReader(buffer.Bytes())
		var got cachePDU
		binary.Read(buf, binary.BigEndian, &got)

		// Directly create PDU
		want := cachePDU{
			Version: version1,
			Ptype:   cacheResponse,
			Session: p.session,
			Length:  8,
		}

		// Compare them
		if !cmp.Equal(got, want) {
			t.Errorf("PDU encoded is not what was expected. Got %+v, Wanted %+v\n", got, want)
		}
	}
}

func TestIpv4PrefixPDU(t *testing.T) {
	type prefixPDU struct {
		Version uint8
		Ptype   uint8
		Zero16  uint16
		Length  uint32
		Flags   uint8
		Min     uint8
		Max     uint8
		Zero8   uint8
		Prefix  [4]byte
		Asn     uint32
	}
	pdus := []struct {
		desc           string
		prefix         [4]byte
		min, max, flag uint8
		asn            uint32
	}{
		{
			desc:   "192.168.0.0/24-/25 AS12345 announce",
			prefix: [4]byte{192, 168, 0, 0},
			min:    24,
			max:    25,
			asn:    12345,
			flag:   announce,
		},
		{
			desc:   "192.168.0.0/24-/25 AS12345 withdraw",
			prefix: [4]byte{192, 168, 0, 0},
			min:    24,
			max:    25,
			asn:    12345,
			flag:   withdraw,
		},
	}

	for _, p := range pdus {

		// Send data to be encoded
		var buffer bytes.Buffer
		pdu := &ipv4PrefixPDU{
			prefix: p.prefix,
			flags:  p.flag,
			min:    p.min,
			max:    p.max,
			asn:    p.asn,
		}
		pdu.serialize(&buffer)

		// Read data back that was written
		buf := bytes.NewReader(buffer.Bytes())
		var got prefixPDU
		binary.Read(buf, binary.BigEndian, &got)

		// Directly create PDU
		want := prefixPDU{
			Version: version1,
			Ptype:   ipv4Prefix,
			Flags:   p.flag,
			Prefix:  p.prefix,
			Min:     p.min,
			Max:     p.max,
			Asn:     p.asn,
			Length:  20,
		}

		// Compare them
		if !cmp.Equal(got, want) {
			t.Errorf("PDU encoded is not what was expected. Got %+v, Wanted %+v\n", got, want)
		}
	}
}

func TestIpv6PrefixPDU(t *testing.T) {
	type prefixPDU struct {
		Version uint8
		Ptype   uint8
		Zero16  uint16
		Length  uint32
		Flags   uint8
		Min     uint8
		Max     uint8
		Zero8   uint8
		Prefix  [16]byte
		Asn     uint32
	}
	pdus := []struct {
		desc           string
		prefix         [16]byte
		min, max, flag uint8
		asn            uint32
	}{
		{
			desc:   "2001:db8::/32-/48 AS123123 announce",
			prefix: [16]byte{32, 1, 13, 184, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			min:    32,
			max:    48,
			asn:    123123,
			flag:   announce,
		},
		{
			desc:   "2001:db8::/32-/48 AS123123 withdraw",
			prefix: [16]byte{32, 1, 13, 184, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			min:    32,
			max:    48,
			asn:    123123,
			flag:   withdraw,
		},
		{
			desc:   "2001:db8:abc:123:12ab:9911:abdc:ef12/128-/128 AS123123 announce",
			prefix: [16]byte{32, 1, 13, 184, 10, 188, 1, 35, 18, 171, 153, 17, 171, 220, 239, 18},
			min:    128,
			max:    128,
			asn:    123123,
			flag:   announce,
		},
		{
			desc:   "2001:db8:abc:123:12ab:9911:abdc:ef12/128-/128 AS123123 withdraw",
			prefix: [16]byte{32, 1, 13, 184, 10, 188, 1, 35, 18, 171, 153, 17, 171, 220, 239, 18},
			min:    128,
			max:    128,
			asn:    123123,
			flag:   withdraw,
		},
	}

	for _, p := range pdus {

		// Send data to be encoded
		var buffer bytes.Buffer
		pdu := &ipv6PrefixPDU{
			prefix: p.prefix,
			flags:  p.flag,
			min:    p.min,
			max:    p.max,
			asn:    p.asn,
		}
		pdu.serialize(&buffer)

		// Read data back that was written
		buf := bytes.NewReader(buffer.Bytes())
		var got prefixPDU
		binary.Read(buf, binary.BigEndian, &got)

		// Directly create PDU
		want := prefixPDU{
			Version: version1,
			Ptype:   ipv6Prefix,
			Flags:   p.flag,
			Prefix:  p.prefix,
			Min:     p.min,
			Max:     p.max,
			Asn:     p.asn,
			Length:  32,
		}

		// Compare them
		if !cmp.Equal(got, want) {
			t.Errorf("PDU encoded is not what was expected. Got %+v, Wanted %+v\n", got, want)
		}
	}
}

func TestEndOfDataPDU(t *testing.T) {
	type eodPDU struct {
		Version uint8
		Ptype   uint8
		Session uint16
		Length  uint32
		Serial  uint32
		Refresh uint32
		Retry   uint32
		Expire  uint32
	}
	pdus := []struct {
		desc    string
		session uint16
		serial  uint32
		refresh uint32
		retry   uint32
		expire  uint32
	}{
		{
			desc:    "test 1",
			session: 1,
			serial:  2,
			refresh: 3,
			retry:   4,
			expire:  5,
		},
		{
			desc: "zero test",
		},
	}
	for _, v := range pdus {
		// Send data to be encoded
		var buffer bytes.Buffer
		pdu := &endOfDataPDU{
			session: v.session,
			serial:  v.serial,
			refresh: v.refresh,
			retry:   v.retry,
			expire:  v.expire,
		}
		pdu.serialize(&buffer)

		// Read data back that was written
		buf := bytes.NewReader(buffer.Bytes())
		var got eodPDU
		binary.Read(buf, binary.BigEndian, &got)

		// Directly create PDU
		want := eodPDU{
			Version: version1,
			Ptype:   endOfData,
			Session: v.session,
			Length:  24,
			Serial:  v.serial,
			Refresh: v.refresh,
			Retry:   v.retry,
			Expire:  v.expire,
		}

		// Compare them
		if !cmp.Equal(got, want) {
			t.Errorf("PDU encoded is not what was expected. Got %+v, Wanted %+v\n", got, want)
		}
	}
}

func TestCacheResetPDU(t *testing.T) {
	type cachePDU struct {
		Version uint8
		Ptype   uint8
		Zero16  uint16
		Length  uint32
	}

	// Send data to be encoded
	var buffer bytes.Buffer
	pdu := &cacheResetPDU{}
	pdu.serialize(&buffer)

	// Read data back that was written
	buf := bytes.NewReader(buffer.Bytes())
	var got cachePDU
	binary.Read(buf, binary.BigEndian, &got)

	// Directly create PDU
	want := cachePDU{
		Version: version1,
		Ptype:   cacheReset,
		Length:  8,
	}

	// Compare them
	if !cmp.Equal(got, want) {
		t.Errorf("PDU encoded is not what was expected. Got %+v, Wanted %+v\n", got, want)
	}
}
