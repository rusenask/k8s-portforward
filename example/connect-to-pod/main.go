package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	portforward "github.com/rusenask/k8s-portforward"
	"k8s.io/client-go/tools/clientcmd"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	pods      = []string{"set-me-to-whatever-is-deployed"}
	namespace = "default"
	podPort   = "port-name"
)

func main() {

	logger := logf.Log.WithName("main")
	logf.SetLogger(zap.Logger())

	kubeConfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")

	restConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		logger.Error(err, "failed to load k8s cfg")
		os.Exit(1)
	}

	for _, pod := range pods {
		hc := &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
					_, port, err := net.SplitHostPort(address)
					if err != nil {
						return nil, err
					}

					return portforward.DialContext(ctx, logger, restConfig, namespace, pod, port)
				},
			},
		}

		// Use created HTTP client to connect to the pod
		var req *http.Request
		req, err = http.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("http://doesntmatter:%s/status", podPort), nil)
		if err != nil {
			logger.Error(err, "failed to prepare HTTP request")
			continue
		}

		resp, err := hc.Do(req)
		if err != nil {
			logger.Error(err, "failed to do HTTP request")
			continue
		}
		bts, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logger.Error(err, "failed to read response body")
			continue
		}
		resp.Body.Close()
		logger.Info("request complete",
			"response", string(bts),
			"status_code", resp.StatusCode,
		)
	}
}
