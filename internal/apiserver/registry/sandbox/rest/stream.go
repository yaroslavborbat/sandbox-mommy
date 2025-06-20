package rest

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
	"k8s.io/apimachinery/pkg/util/proxy"
	"k8s.io/apiserver/pkg/registry/rest"
)

type wsStreamReader struct {
	conn *websocket.Conn
}

func (r *wsStreamReader) Read(p []byte) (int, error) {
	_, msg, err := r.conn.ReadMessage()
	if err != nil {
		return 0, err
	}
	return copy(p, msg), nil
}

type wsStreamWriter struct {
	conn *websocket.Conn
}

func (w *wsStreamWriter) Write(p []byte) (int, error) {
	err := w.conn.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func isWebSocketRequest(req *http.Request) bool {
	return strings.ToLower(req.Header.Get("Upgrade")) == "websocket" &&
		strings.Contains(strings.ToLower(req.Header.Get("Connection")), "upgrade")
}

const PerConnectionBandwidthLimitBytesPerSec = 1024

func newThrottledUpgradeAwareProxyHandler(location *url.URL, transport http.RoundTripper, wrapTransport, upgradeRequired bool, responder rest.Responder) http.Handler {
	handler := proxy.NewUpgradeAwareHandler(location, transport, wrapTransport, upgradeRequired, proxy.NewErrorResponder(responder))
	handler.MaxBytesPerSec = PerConnectionBandwidthLimitBytesPerSec
	return handler
}
