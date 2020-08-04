package main

import (
	"fmt"
	"net/http"

	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/config"
	"github.com/sirupsen/logrus"
)

// makeRolloutHandler creates a request handler to perform a rollout process.
func makeRolloutHandler(logger *logrus.Logger, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		services, err := getTargetedServices(ctx, logger, cfg.Targets)
		if err != nil {
			logger.Errorf("failed to get targeted services: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "failed to retrieved services %v", err)
			return
		}
		if len(services) == 0 {
			logger.Warn("no service matches the targets")
		}

		// TODO(gvso): Filter "fatal" errors from "no-bad" errors (e.g. no
		// requests in interval when getting metrics).
		errs := runRollouts(ctx, logger, services, cfg.Strategy)
		errsStr := rolloutErrsToString(errs)
		if len(errs) != 0 {
			msg := fmt.Sprintf("there were %d errors: %s", len(errs), errsStr)
			logger.Warn(msg)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, msg)
		}
	}
}
