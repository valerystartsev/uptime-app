package monitorurl

import (
	"net"
	"net/url"
	"strings"
)

// Key identifies equivalent HTTP addresses without altering the stored URL.
func Key(value string) string {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Hostname() == "" {
		return value
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	host := strings.ToLower(parsed.Hostname())
	port := parsed.Port()
	if (parsed.Scheme == "http" && port == "80") || (parsed.Scheme == "https" && port == "443") {
		port = ""
	}
	if port != "" {
		parsed.Host = net.JoinHostPort(host, port)
	} else if strings.Contains(host, ":") {
		parsed.Host = "[" + host + "]"
	} else {
		parsed.Host = host
	}
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	return parsed.String()
}
