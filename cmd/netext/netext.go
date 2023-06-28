// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"net/netip"
	"sort"

	"tailscale.com/ipn"
	"tailscale.com/tailcfg"
	"tailscale.com/types/netmap"
)

type ExitStatus uint8

const (
	// No exit node selected.
	ExitNone ExitStatus = iota
	// Exit node selected and exists, but is offline or missing.
	ExitOffline
	// Exit node selected and online.
	ExitOnline
)

type Peer struct {
	Label  string
	Online bool
	ID     tailcfg.StableNodeID
}

type BackendState struct {
	Prefs        *ipn.Prefs
	State        ipn.State
	NetworkMap   *netmap.NetworkMap
	LostInternet bool
	// Exits are the peers that can act as exit node.
	Exits []Peer
	// ExitState describes the state of our exit node.
	ExitStatus ExitStatus
	// Exit is our current exit node, if any.
	Exit Peer
}

func (s *BackendState) updateExitNodes() {
	s.ExitStatus = ExitNone
	var exitID tailcfg.StableNodeID
	if p := s.Prefs; p != nil {
		exitID = p.ExitNodeID
		if exitID != "" {
			s.ExitStatus = ExitOffline
		}
	}
	hasMyExit := exitID == ""
	s.Exits = nil
	var peers []*tailcfg.Node
	if s.NetworkMap != nil {
		peers = s.NetworkMap.Peers
	}
	for _, p := range peers {
		canRoute := false
		for _, r := range p.AllowedIPs {
			if r == netip.MustParsePrefix("0.0.0.0/0") || r == netip.MustParsePrefix("::/0") {
				canRoute = true
				break
			}
		}
		myExit := p.StableID == exitID
		hasMyExit = hasMyExit || myExit
		exit := Peer{
			Label:  p.DisplayName(true),
			Online: canRoute,
			ID:     p.StableID,
		}
		if myExit {
			s.Exit = exit
			if canRoute {
				s.ExitStatus = ExitOnline
			}
		}
		if canRoute || myExit {
			s.Exits = append(s.Exits, exit)
		}
	}
	sort.Slice(s.Exits, func(i, j int) bool {
		return s.Exits[i].Label < s.Exits[j].Label
	})
	if !hasMyExit {
		// Insert node missing from netmap.
		s.Exit = Peer{Label: "Unknown device", ID: exitID}
		s.Exits = append([]Peer{s.Exit}, s.Exits...)
	}
}

var (
	state     BackendState
	signingIn bool
)

func runBackend() error {

	notifications := make(chan ipn.Notify, 1)
	startErr := make(chan error)
	// Start from a goroutine to avoid deadlock when Start
	// calls the callback.
	go func() {
		startErr <- b.Start(func(n ipn.Notify) {
			notifications <- n
		})
	}()
	for {
		select {
		case err := <-startErr:
			if err != nil {
				return err
			}
		case n := <-notifications:
			if p := n.Prefs; p != nil && n.Prefs.Valid() {
				first := state.Prefs == nil
				state.Prefs = p.AsStruct()
				state.updateExitNodes()
				if first {
					state.Prefs.Hostname = "TODO" //TODO: get host name by NE API
					go b.backend.SetPrefs(state.Prefs)
				}
			}
			if s := n.State; s != nil {
				oldState := state.State
				state.State = *s

				// Stop VPN if we logged out.
				if oldState > ipn.Stopped && state.State <= ipn.Stopped {
					// TODO, notify app to stop VPN, maybe NE can just all stopTunnel directly?
				}
			}
			if u := n.BrowseToURL; u != nil {
				signingIn = false
				//a.setURL(*u)
				// TODO, call swift to open url
			}
			if m := n.NetMap; m != nil {
				state.NetworkMap = m
				state.updateExitNodes()
			}
		}
	}
}

func main() {}
