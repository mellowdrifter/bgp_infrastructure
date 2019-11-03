package main

const (
	// PDU Types
	serialNotify  uint8 = 0
	serialQuery   uint8 = 1
	resetQuery    uint8 = 2
	cacheResponse uint8 = 3
	ipv4Prefix    uint8 = 4
	ipv6Prefix    uint8 = 6
	endOfData     uint8 = 7
	cacheReset    uint8 = 8
	routerKey     uint8 = 9
	errorReport   uint8 = 10

	// protocol versions
	version0 uint8 = 0
	version1 uint8 = 1

	zeroUint16    uint16 = 0
	length8Uint8  uint8  = 8
	length20Uint8 uint8  = 20
)
