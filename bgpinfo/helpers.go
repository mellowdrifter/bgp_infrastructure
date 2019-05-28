package main

import (
	pb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpinfo"
)

// A struct to hold the AS information altogether
type bgpStat struct {
	time              uint64
	v4Count, v6Count  uint32
	peersConfigured   uint8
	peers6Configured  uint8
	peersUp, peers6Up uint8
	v4Total, v6Total  uint32
}

// bgpUpdate holds all the information required for an update
type bgpUpdate struct {
	time                                uint64
	v4Count, v6Count                    uint32
	v4Total, v6Total                    uint32
	peersConfigured                     uint32
	peers6Configured                    uint32
	peersUp, peers6Up                   uint32
	tweet                               bool
	as4, as6, as10                      uint32
	as4Only, as6Only                    uint32
	asBoth                              uint32
	largeC4, largeC6                    uint32
	memTable, memTotal                  string
	memProto, memAttr                   string
	memTable6, memTotal6                string
	memProto6, memAttr6                 string
	roavalid4, roainvalid4, roaunknown4 uint32
	roavalid6, roainvalid6, roaunknown6 uint32
	v4_23, v4_22, v4_21, v4_20, v4_19   uint32
	v4_18, v4_17, v4_16, v4_15, v4_14   uint32
	v4_13, v4_12, v4_11, v4_10, v4_09   uint32
	v4_08, v6_48, v6_47, v6_46, v6_45   uint32
	v6_44, v6_43, v6_42, v6_41, v6_40   uint32
	v6_39, v6_38, v6_37, v6_36, v6_35   uint32
	v6_34, v6_33, v6_32, v6_31, v6_30   uint32
	v6_29, v6_28, v6_27, v6_26, v6_25   uint32
	v6_24, v6_23, v6_22, v6_21, v6_20   uint32
	v6_19, v6_18, v6_17, v6_16, v6_15   uint32
	v6_14, v6_13, v6_12, v6_11, v6_10   uint32
	v6_09, v6_08, v4_24                 uint32
}

func repack(v *pb.Values) *bgpUpdate {
	// While we receive this information in a protobuf, the
	// format needs to be adjusted a bit to insert into the
	// database later.
	as := v.GetAsCount()
	mem := v.GetMemUse()
	mask := v.GetMasks()
	p := v.GetPrefixCount()
	roa := v.GetRoas()
	update := &bgpUpdate{
		time:             v.GetTime(),
		v4Count:          p.GetActive_4(),
		v6Count:          p.GetActive_6(),
		v4Total:          p.GetTotal_4(),
		v6Total:          p.GetTotal_6(),
		as4:              as.GetAs4(),
		as6:              as.GetAs6(),
		as10:             as.GetAs10(),
		as4Only:          as.GetAs4Only(),
		as6Only:          as.GetAs6Only(),
		asBoth:           as.GetAsBoth(),
		largeC4:          v.GetLargeCommunity().GetC4(),
		largeC6:          v.GetLargeCommunity().GetC6(),
		peersConfigured:  v.GetPeers().GetPeerCount_4(),
		peersUp:          v.GetPeers().GetPeerUp_4(),
		peers6Configured: v.GetPeers().GetPeerCount_6(),
		peers6Up:         v.GetPeers().GetPeerUp_6(),
		roavalid4:        roa.GetV4Valid(),
		roainvalid4:      roa.GetV4Invalid(),
		roaunknown4:      roa.GetV4Unknown(),
		roavalid6:        roa.GetV6Valid(),
		roainvalid6:      roa.GetV6Invalid(),
		roaunknown6:      roa.GetV6Unknown(),
		v4_08:            mask.GetV4_08(),
		v4_09:            mask.GetV4_09(),
		v4_10:            mask.GetV4_10(),
		v4_11:            mask.GetV4_11(),
		v4_12:            mask.GetV4_12(),
		v4_13:            mask.GetV4_13(),
		v4_14:            mask.GetV4_14(),
		v4_15:            mask.GetV4_15(),
		v4_16:            mask.GetV4_16(),
		v4_17:            mask.GetV4_17(),
		v4_18:            mask.GetV4_18(),
		v4_19:            mask.GetV4_19(),
		v4_20:            mask.GetV4_20(),
		v4_21:            mask.GetV4_21(),
		v4_22:            mask.GetV4_22(),
		v4_23:            mask.GetV4_23(),
		v4_24:            mask.GetV4_24(),
		v6_08:            mask.GetV6_08(),
		v6_09:            mask.GetV6_09(),
		v6_10:            mask.GetV6_10(),
		v6_11:            mask.GetV6_11(),
		v6_12:            mask.GetV6_12(),
		v6_13:            mask.GetV6_13(),
		v6_14:            mask.GetV6_14(),
		v6_15:            mask.GetV6_15(),
		v6_16:            mask.GetV6_16(),
		v6_17:            mask.GetV6_17(),
		v6_18:            mask.GetV6_18(),
		v6_19:            mask.GetV6_19(),
		v6_20:            mask.GetV6_20(),
		v6_21:            mask.GetV6_21(),
		v6_22:            mask.GetV6_22(),
		v6_23:            mask.GetV6_23(),
		v6_24:            mask.GetV6_24(),
		v6_25:            mask.GetV6_25(),
		v6_26:            mask.GetV6_26(),
		v6_27:            mask.GetV6_27(),
		v6_28:            mask.GetV6_28(),
		v6_29:            mask.GetV6_29(),
		v6_30:            mask.GetV6_30(),
		v6_31:            mask.GetV6_31(),
		v6_32:            mask.GetV6_32(),
		v6_33:            mask.GetV6_33(),
		v6_34:            mask.GetV6_34(),
		v6_35:            mask.GetV6_35(),
		v6_36:            mask.GetV6_36(),
		v6_37:            mask.GetV6_37(),
		v6_38:            mask.GetV6_38(),
		v6_39:            mask.GetV6_39(),
		v6_40:            mask.GetV6_40(),
		v6_41:            mask.GetV6_41(),
		v6_42:            mask.GetV6_42(),
		v6_43:            mask.GetV6_43(),
		v6_44:            mask.GetV6_44(),
		v6_45:            mask.GetV6_45(),
		v6_46:            mask.GetV6_46(),
		v6_47:            mask.GetV6_47(),
		v6_48:            mask.GetV6_48(),
	}

	for _, m := range mem {
		if m.GetFamily() == pb.AddressFamily_IPV4 {
			update.memTable = m.GetMemstats().GetTables()
			update.memProto = m.GetMemstats().GetProtocols()
			update.memAttr = m.GetMemstats().GetAttributes()
			update.memTotal = m.GetMemstats().GetTotal()
		}
		if m.GetFamily() == pb.AddressFamily_IPV6 {
			update.memTable6 = m.GetMemstats().GetTables()
			update.memProto6 = m.GetMemstats().GetProtocols()
			update.memAttr6 = m.GetMemstats().GetAttributes()
			update.memTotal6 = m.GetMemstats().GetTotal()
		}
	}

	return update
}
