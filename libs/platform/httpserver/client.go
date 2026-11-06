package httpserver

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

const forwardedForHeader = "X-Forwarded-For"

func ClientAddress(r *http.Request) (netip.Addr, bool) {
	if forwarded, found := lastForwardedAddress(r.Header); found {
		return forwarded, true
	}
	return remoteAddress(r.RemoteAddr)
}

func lastForwardedAddress(header http.Header) (netip.Addr, bool) {
	lines := header.Values(forwardedForHeader)
	if len(lines) == 0 {
		return netip.Addr{}, false
	}
	entries := strings.Split(lines[len(lines)-1], ",")
	return parseAddress(entries[len(entries)-1])
}

func remoteAddress(remote string) (netip.Addr, bool) {
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		return parseAddress(remote)
	}
	return parseAddress(host)
}

func parseAddress(raw string) (netip.Addr, bool) {
	address, err := netip.ParseAddr(strings.TrimSpace(raw))
	if err != nil {
		return netip.Addr{}, false
	}
	return address.Unmap().WithZone(""), true
}
