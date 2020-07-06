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
	"time"

	"github.com/GoogleCloudPlatform/cloud-run-release-operator/internal/run"
	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/config"
	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/rollout"
	log "github.com/sirupsen/logrus"
)

func main() {
	config := &config.Config{
		Metadata: &config.Metadata{
			Project: "gtvo-test",
			Service: "hello",
			Region:  "us-east1",
		},
		Rollout: &config.Rollout{
			Steps: []int64{5, 30, 80},
		},
	}
	log := log.New()

	client, err := run.NewAPIClient(context.Background(), config.Metadata.Region)
	if err != nil {
		log.Fatalf("could not initilize Cloud Run client: %v\n", err)
	}
	roll := rollout.New(client, config, log)

	svc, err := roll.Manage()
	if err != nil {
		log.Printf("Rollout failed: %v\n", err)
	}

	if svc != nil {
		log.Println("Rollout process succeeded")
	}

	interval := time.Duration(config.Rollout.Interval)
	time.Sleep(interval * time.Second)
}
