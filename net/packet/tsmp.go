// Copyright (c) 2021 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// TSMP is our ICMP-like "Tailscale Message Protocol" for signaling
// Tailscale-specific messages between nodes. It uses IP protocol 99
// (reserved for "any private encryption scheme") underneath
// Wireguard's normal encryption between peers and never hits the host
// network stack.

package packet

import (
	"encoding/binary"
	"errors"

	"inet.af/netaddr"
)

// TailscaleRejectedHeader is a TSMP message that says that one
// Tailscale node has rejected the connection from another. Unlike a
// TCP RST, this includes a reason.
type TailscaleRejectedHeader struct {
	Src    netaddr.IPPort        // initiator's address
	Dst    netaddr.IPPort        // the destination that failed
	Proto  IPProto               // proto that was rejected (TCP or UDP)
	Reason TailscaleRejectReason // why the connection was rejected
}

type TSMPType uint8

const (
	TSMPTypeRejectedConn TSMPType = '!'
)

type TailscaleRejectReason byte

const (
	RejectedDueToACLs      TailscaleRejectReason = 'A'
	RejectedDueToShieldsUp TailscaleRejectReason = 'S'
)

func (h TailscaleRejectedHeader) Len() int {
	var ipHeaderLen int
	if h.Src.IP.Is4() {
		ipHeaderLen = ip4HeaderLength
	} else if h.Src.IP.Is6() {
		ipHeaderLen = ip6HeaderLength
	}
	return ipHeaderLen +
		1 + // TSMPType byte
		1 + // IPProto byte
		1 + // TailscaleRejectReason byte
		2*2 // 2 uint16 ports
}

func (h TailscaleRejectedHeader) Marshal(buf []byte) error {
	if len(buf) < h.Len() {
		return errSmallBuffer
	}
	if len(buf) > maxPacketLength {
		return errLargePacket
	}
	if h.Src.IP.Is4() {
		iph := IP4Header{
			IPProto: TSMP,
			Src:     h.Dst.IP, // reversed
			Dst:     h.Src.IP, // reversed
		}
		iph.Marshal(buf)
		buf = buf[ip4HeaderLength:]
	} else if h.Src.IP.Is6() {
		iph := IP6Header{
			IPProto: TSMP,
			Src:     h.Dst.IP, // reversed
			Dst:     h.Src.IP, // reversed
		}
		iph.Marshal(buf)
		buf = buf[ip6HeaderLength:]
	} else {
		return errors.New("bogus src IP")
	}
	buf[0] = byte(TSMPTypeRejectedConn)
	buf[1] = byte(h.Proto)
	buf[2] = byte(h.Reason)
	binary.BigEndian.PutUint16(buf[3:5], h.Src.Port)
	binary.BigEndian.PutUint16(buf[5:7], h.Dst.Port)
	return nil
}

// parseTSMPPayload parses a TSMP packet's payload (after the IPv4 or
// IPv6 header) into pp.
func parseTSMPPayload(pp *Parsed, buf []byte) {
	if len(buf) == 0 {
		return
	}
	switch buf[0] {
	case byte(TSMPTypeRejectedConn):
		if len(buf) < 7 {
			return
		}
		pp.Src.Port = binary.BigEndian.Uint16(buf[3:5])
		pp.Dst.Port = binary.BigEndian.Uint16(buf[5:7])
	}
}
