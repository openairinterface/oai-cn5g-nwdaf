/*
 * SPDX-License-Identifier: LicenseRef-CSSL-1.0
 */

/*
 * Notification delivery over HTTP/2 or HTTP/1.1.
 *
 * Consumers inside the 5GC (the NEF relaying analytics to an untrusted AF)
 * speak the SBI's cleartext HTTP/2 only, while an AF reached through the NBI
 * may speak HTTP/1.1 only. Neither says which in the notification URI, so try
 * h2c first and fall back to HTTP/1.1, then remember per host what worked.
 */

package events

import (
	"bytes"
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/net/http2"
)

var (
	h2cClient = &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, network, addr)
			},
		},
	}
	h1Client = &http.Client{Timeout: 5 * time.Second}

	// host -> true when it answered over h2c, false when it needed HTTP/1.1
	hostSpeaksH2c sync.Map
)

func postNotification(uri string, body []byte) error {
	u, err := url.Parse(uri)
	if err != nil {
		return err
	}
	post := func(c *http.Client) error {
		resp, err := c.Post(uri, "application/json", bytes.NewReader(body))
		if err == nil {
			resp.Body.Close()
		}
		return err
	}
	if v, ok := hostSpeaksH2c.Load(u.Host); ok {
		if v.(bool) {
			return post(h2cClient)
		}
		return post(h1Client)
	}
	if err = post(h2cClient); err == nil {
		hostSpeaksH2c.Store(u.Host, true)
		return nil
	}
	if err = post(h1Client); err == nil {
		hostSpeaksH2c.Store(u.Host, false)
	}
	return err
}
