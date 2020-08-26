package cloudrun

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/container/v1"
)

// newGKEClient initializes a client for GKE cluster.
func newGKEClient(ctx context.Context, project, zone, clusterName string) (*http.Client, string, error) {
	tokenSource, err := google.DefaultTokenSource(ctx, compute.CloudPlatformScope)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to get token source")
	}

	httpClient := oauth2.NewClient(ctx, tokenSource)
	containerService, err := container.New(httpClient)
	if err != nil {
		return nil, "", fmt.Errorf("could not create client for Google Kubernetes Engine: %v", err)
	}

	// TODO: handle regional clusters
	cluster, err := containerService.Projects.Zones.Clusters.Get(project, zone, clusterName).Context(ctx).Do()
	if err != nil {
		return nil, "", errors.Wrapf(err, "failed to get cluster %q in project %q, zone %q", clusterName, project, zone)
	}

	hClient, err := newGKEHTTPClient(cluster, tokenSource)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to initialize HTTP client")
	}
	return hClient, cluster.Endpoint, nil
}

func newGKEHTTPClient(cluster *container.Cluster, tokenSource oauth2.TokenSource) (*http.Client, error) {
	tlsCfg, err := gkeTLSConfig(cluster)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get TLS configuration")
	}

	hTransport := http.DefaultTransport.(*http.Transport).Clone()
	hTransport.TLSClientConfig = tlsCfg
	oauth2Transport := &oauth2.Transport{
		Base:   hTransport,
		Source: tokenSource,
	}
	return &http.Client{Transport: oauth2Transport}, nil
}

func gkeTLSConfig(cluster *container.Cluster) (*tls.Config, error) {
	caCert, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClusterCaCertificate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode cluster certificate")
	}

	// CA Cert from kube master
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(caCert))

	// Setup TLS config
	tlsConfig := &tls.Config{
		RootCAs: caCertPool,
	}

	return tlsConfig, nil
}
