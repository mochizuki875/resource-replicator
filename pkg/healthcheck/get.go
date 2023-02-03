package healthcheck

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/jnytnai0613/resource-replicator/pkg/kubeconfig"
)

// Throw a request to kube-apiserver on the remote Kubernetes cluster
// and check if it can communicate successfully.
// The following function throws a request to the "livez" API endpoint.
// https://kubernetes.io/docs/reference/using-api/health-checks/
func HealthChecks(target kubeconfig.RemoteAPIServer) error {
	client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				// Since no communication is made using certificates,　certificate
				// verification is skipped.
				InsecureSkipVerify: true,
			},
		},
		// If no response is received within 2 seconds,
		// the communication is considered to have failed.
		Timeout: 2 * time.Second,
	}

	u := fmt.Sprintf("%s%s", target.Endpoint, "/livez")
	resp, err := client.Get(u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	byteArray, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(byteArray))

	return nil
}
