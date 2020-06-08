# Kubernetes HTTP client transport via API server

A custom HTTP transport to connect to any pod from inside/outside of the cluster.

## Use case 

You have a Deployment, DaemonSet, StatefulSet and need to connect to an individual pod to collect Prometheus metrics, status, etc. 

Best part: you can use this from both outside of the cluster and from inside too. Good for testing/development!

## Usage

```golang
import (
  // other libs commented out

	// import the lib
	portforward "github.com/rusenask/k8s-portforward"
)

var (
	podName   = "set-me-to-whatever-is-deployed"
	namespace = "default"
	podPort   = "5555"
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

  hc := &http.Client{
    Transport: &http.Transport{
      DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
        _, port, err := net.SplitHostPort(address)
        if err != nil {
          return nil, err
        }

        return portforward.DialContext(ctx, logger, restConfig, namespace, podName, port)
      },
    },
  }

  // Use created HTTP client to connect to the pod
  var req *http.Request
  req, err = http.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("http://doesntmatter:%s/metrics", podPort), nil)
  if err != nil {
    logger.Error(err, "failed to prepare HTTP request")
    continue
  }

  resp, err := hc.Do(req)
  if err != nil {
    logger.Error(err, "failed to do HTTP request")
    continue
  }
  // process response
	
}
```

## Thank you

Thanks to https://github.com/Azure/ARO-RP for giving ideas how to connect to k8s pods based on their implementation when connecting to pods in OpenShift