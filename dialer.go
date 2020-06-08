package portforward

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/apimachinery/pkg/util/httpstream/spdy"
	"k8s.io/client-go/rest"
)

// dialSpdy connects to the specified path on the API server of k8s and
// negotiates SPDY
func dialSpdy(ctx context.Context, restconfig *rest.Config, path string) (httpstream.Connection, error) {

	// 1. Connect to the API server via private endpoint (and via proxy in
	//    development mode)
	clusterURL, err := url.Parse(restconfig.Host)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse Kubernetes API server host")
	}

	hostPort := clusterURL.Host
	// net.Dial
	rawConn, err := net.Dial("tcp", hostPort)
	if err != nil {
		return nil, errors.Wrap(err, "failed to dial API server")
	}

	// 2. Negotiate TLS
	tlsConfig, err := rest.TLSConfigFor(restconfig)
	if err != nil {
		rawConn.Close()
		return nil, err
	}
	tlsConfig.ServerName = clusterURL.Hostname()

	tlsConn := tls.Client(rawConn, tlsConfig)
	err = tlsConn.Handshake()
	if err != nil {
		tlsConn.Close()
		return nil, err
	}

	// 3. Issue an HTTP POST request to the specified path
	req, err := http.NewRequest(http.MethodPost, path, nil)
	if err != nil {
		tlsConn.Close()
		return nil, err
	}
	req.Header.Add(httpstream.HeaderConnection, httpstream.HeaderUpgrade)
	req.Header.Add(httpstream.HeaderUpgrade, spdy.HeaderSpdy31)

	err = req.Write(tlsConn)
	if err != nil {
		tlsConn.Close()
		return nil, err
	}

	// 4. Validate the response
	resp, err := http.ReadResponse(bufio.NewReader(tlsConn), req)
	if err != nil {
		tlsConn.Close()
		return nil, err
	}

	if resp.StatusCode != http.StatusSwitchingProtocols {
		tlsConn.Close()
		return nil, fmt.Errorf("unexpected http status code %d", resp.StatusCode)
	}

	if resp.Header.Get(httpstream.HeaderConnection) != httpstream.HeaderUpgrade {
		tlsConn.Close()
		return nil, fmt.Errorf("unexpected http header %s: %s", httpstream.HeaderConnection, resp.Header.Get(httpstream.HeaderConnection))
	}

	if resp.Header.Get(httpstream.HeaderUpgrade) != spdy.HeaderSpdy31 {
		tlsConn.Close()
		return nil, fmt.Errorf("unexpected http header %s: %s", httpstream.HeaderUpgrade, resp.Header.Get(httpstream.HeaderUpgrade))
	}

	// 5. Negotiate SPDY
	spdyConn, err := spdy.NewClientConnection(tlsConn)
	if err != nil {
		tlsConn.Close()
		return nil, err
	}

	return spdyConn, nil
}
