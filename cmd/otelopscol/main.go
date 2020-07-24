// Copyright 2020, Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/service"
	"go.opentelemetry.io/collector/service/builder"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-collector/internal/version"
)

func main() {
	factories, err := components()
	if err != nil {
		log.Fatalf("failed to build default components: %v", err)
	}

	info := service.ApplicationStartInfo{
		ExeName:  "google-cloudops-opentelemetry-collector",
		LongName: "OpenTelemetry Cloud Operations Collector",
		Version:  version.Version,
		GitHash:  version.GitHash,
	}

	params := service.Parameters{Factories: factories, ConfigFactory: loadConfig, ApplicationStartInfo: info}

	if err := run(params); err != nil {
		log.Fatal(err)
	}
}

func runInteractive(params service.Parameters) error {
	app, err := service.New(params)
	if err != nil {
		return fmt.Errorf("failed to construct the application: %w", err)
	}

	err = app.Start()
	if err != nil {
		return fmt.Errorf("application run finished with error: %w", err)
	}

	return nil
}

func loadConfig(v *viper.Viper, factories config.Factories) (*configmodels.Config, error) {
	file := builder.GetConfigFile()
	if file == "" {
		return nil, errors.New("config file not specified")
	}

	// load user config file
	v.SetConfigFile(file)
	err := v.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("error loading config file %q: %v", file, err)
	}

	// load cloud monitoring agent config file from same directory and merge
	stackdriverConfigFile := fmt.Sprintf("%v/config-cloud-monitoring-agent.yaml", filepath.Dir(file))
	v.SetConfigFile(stackdriverConfigFile)
	err = v.MergeInConfig()
	if err != nil {
		return nil, fmt.Errorf("error loading config file %q: %v", stackdriverConfigFile, err)
	}

	return config.Load(v, factories)
}
