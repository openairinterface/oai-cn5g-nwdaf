/*
 * SPDX-License-Identifier: LicenseRef-CSSL-1.0
 */

package events

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// A notification must reach both an HTTP/1.1-only AF and an h2c-only NEF, and
// the protocol that worked is remembered per host.
func TestPostNotificationReachesH1AndH2c(t *testing.T) {
	got := make(chan string, 2)
	record := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PRI" { // the h2c preface, as an HTTP/1.1 server parses it
			http.Error(w, "", http.StatusBadRequest)
			return
		}
		b, _ := io.ReadAll(r.Body)
		got <- r.Proto + " " + string(b)
		w.WriteHeader(http.StatusNoContent)
	}

	// HTTP/1.1 only, like the uvicorn AF: an h2c preface is not understood.
	h1 := httptest.NewUnstartedServer(http.HandlerFunc(record))
	h1.Config.Protocols = new(http.Protocols)
	h1.Config.Protocols.SetHTTP1(true)
	h1.Start()
	defer h1.Close()
	// h2c only: reject anything that is not HTTP/2.
	h2 := httptest.NewServer(h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor != 2 {
			http.Error(w, "h2c only", http.StatusHTTPVersionNotSupported)
			return
		}
		record(w, r)
	}), &http2.Server{}))
	defer h2.Close()

	for _, c := range []struct {
		uri, wantProto string
		wantH2c        bool
	}{{h1.URL + "/notify", "HTTP/1.1", false}, {h2.URL + "/notify", "HTTP/2.0", true}} {
		if err := postNotification(c.uri, []byte(`{"event":"QOS_SUSTAINABILITY"}`)); err != nil {
			t.Fatalf("%s: %v", c.uri, err)
		}
		if g := <-got; g != c.wantProto+` {"event":"QOS_SUSTAINABILITY"}` {
			t.Fatalf("%s: got %q", c.uri, g)
		}
		u, _ := url.Parse(c.uri)
		if v, _ := hostSpeaksH2c.Load(u.Host); v != c.wantH2c {
			t.Fatalf("%s: cached h2c=%v, want %v", c.uri, v, c.wantH2c)
		}
	}
}
