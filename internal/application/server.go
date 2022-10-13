package application

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/launchdarkly/ld-relay/v6/config"

	"github.com/launchdarkly/go-sdk-common/v3/ldlog"
)

// StartHTTPServer starts the server, with or without TLS. It returns immediately, starting the server
// on a separate goroutine; if the server fails to start up, it sends an error to the error channel.
func StartHTTPServer(
	port int,
	handler http.Handler,
	tlsEnabled bool,
	tlsCertFile, tlsKeyFile string,
	tlsMinVersion uint16,
	loggers ldlog.Loggers,
) (*http.Server, <-chan error) {
	srv := &http.Server{ //nolint:gosec // known issue sc-168042
		Addr:    fmt.Sprintf(":%d", port),
		Handler: handler,
	}

	if tlsEnabled && tlsMinVersion != 0 {
		srv.TLSConfig = &tls.Config{ //nolint:gosec // yes, we know we are potentially allowing TLS <1.2 if the customer demanded it.
			MinVersion: tlsMinVersion,
		}
	}

	errCh := make(chan error)

	go func() {
		var err error
		loggers.Infof("Starting server listening on port %d\n", port)
		if tlsEnabled {
			message := "TLS enabled for server"
			if tlsMinVersion != 0 {
				message += fmt.Sprintf(" (minimum TLS version: %s)", config.NewOptTLSVersion(tlsMinVersion).String())
			}
			loggers.Info(message)
			err = srv.ListenAndServeTLS(tlsCertFile, tlsKeyFile)
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil {
			errCh <- err
		}
	}()

	return srv, errCh
}
