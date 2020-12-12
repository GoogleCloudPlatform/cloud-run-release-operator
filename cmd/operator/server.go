package main

import (
	"fmt"
	"net/http"

	"github.com/GoogleCloudPlatform/cloud-run-release-manager/internal/config"
	ps "github.com/GoogleCloudPlatform/cloud-run-release-manager/internal/notification/pubsub"
	"github.com/sirupsen/logrus"
)

// makeRolloutHandler creates a request handler to perform a rollout process.
func makeRolloutHandler(logger *logrus.Logger, cfg *config.Config, pubsub ps.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		// TODO(gvso): Handle all the strategies.
		errs := runRollouts(ctx, logger, cfg.Strategies[0], pubsub)
		errsStr := rolloutErrsToString(errs)
		if len(errs) != 0 {
			msg := fmt.Sprintf("there were %d errors: \n%s", len(errs), errsStr)
			logger.Warn(msg)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, msg)
		}
	}
}
