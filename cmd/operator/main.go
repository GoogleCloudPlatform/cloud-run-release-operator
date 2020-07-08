// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"flag"
	"io/ioutil"
	"os"
	"time"

	"github.com/GoogleCloudPlatform/cloud-run-release-operator/internal/run"
	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/config"
	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/rollout"
	stackdriver "github.com/TV4/logrus-stackdriver-formatter"
	isatty "github.com/mattn/go-isatty"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

var (
	flCLI        bool
	flConfigFile string
	flHTTPAddr   string
)

func init() {
	flag.BoolVar(&flCLI, "cli", false, "run as CLI application to manage rollout in intervals")
	flag.StringVar(&flConfigFile, "file", "", "the configuration file for the rollout in CLI mode")
	flag.StringVar(&flHTTPAddr, "http-addr", "", "listen on http portrun on request (e.g. :8080)")
	flag.Parse()

	if !flCLI && flHTTPAddr == "" {
		log.Fatal("one of -cli or -http-addr must be set")
	}

	if flCLI && flHTTPAddr != "" {
		log.Fatal("only one of -cli or -http-addr can be used")
	}

	if flCLI && flConfigFile == "" {
		log.Fatal("use -file to set configuration file")
	}
}

func main() {
	logger := log.New()
	if !isatty.IsTerminal(os.Stdout.Fd()) {
		logger.Formatter = stackdriver.NewFormatter(
			stackdriver.WithService("cloud-run-release-operator"),
		)
	}

	if flCLI {
		runCLI(logger, flConfigFile)
	}
}

func runCLI(logger *logrus.Logger, file string) {
	fileData, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatalf("config file could not be read: %v", err)
	}
	config, err := config.Decode(fileData, true)
	if err != nil {
		log.Fatalf("invalid config file: %v", err)
	}

	client, err := run.NewAPIClient(context.Background(), config.Metadata.Region)
	if err != nil {
		log.Fatalf("could not initilize Cloud Run client: %v", err)
	}
	roll := rollout.New(client, config).WithLogger(logger)

	for {
		changed, err := roll.Rollout()
		if err != nil {
			log.Infof("Rollout failed: %v", err)
		}
		if changed {
			log.Info("Rollout process succeeded")
		}

		interval := time.Duration(config.Rollout.Interval)
		time.Sleep(interval * time.Second)
	}
}
