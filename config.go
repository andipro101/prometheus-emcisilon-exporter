/*
Copyright 2018 Adobe
All Rights Reserved.

NOTICE: Adobe permits you to use, modify, and distribute this file in
accordance with the terms of the Adobe license agreement accompanying
it. If you have received this file from a source other than Adobe,
then your use, modification, or distribution of it requires the prior
written permission of Adobe.
*/
package main

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// ClusterConfig holds the configuration for a single Isilon cluster.
type ClusterConfig struct {
	Host        string `yaml:"host"`
	Port        string `yaml:"port"`
	Username    string `yaml:"username"`
	PasswordEnv string `yaml:"password_env"`
	Site        string `yaml:"site"`
	QuotaOnly   bool   `yaml:"quota_only"`
	QuotaRetry  int64  `yaml:"quota_retry"`
}

// Config holds the top-level exporter configuration.
type Config struct {
	Clusters []ClusterConfig `yaml:"clusters"`
}

// loadConfig reads and parses the YAML config file at the given path.
func loadConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read config file %s: %s", path, err)
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("could not parse config file %s: %s", path, err)
	}
	return cfg, nil
}
