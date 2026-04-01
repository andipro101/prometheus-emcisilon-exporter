/*
Copyright 2018 Adobe
All Rights Reserved.

NOTICE: Adobe permits you to use, modify, and distribute this file in
accordance with the terms of the Adobe license agreement accompanying
it. If you have received this file from a source other than Adobe,
then your use, modification, or distribution of it requires the prior
written permission of Adobe.
*/

package collector

import (
	"github.com/adobe/prometheus-emcisilon-exporter/isiclient"
	"github.com/hpanike/goisilon"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

// IsilonCluster struct contains all the connection info and an instantiated client connection to the cluster.
type IsilonCluster struct {
	Host        string
	Name        string
	Port        string
	Username    string
	Site        string
	PasswordEnv string
	QuotaOnly   bool
	Quotas      Quotas
	Client      *goisilon.Client
}

// Quotas struct contains information for quota-only collections.
type Quotas struct {
	Count  int64
	Errors int64
	Err    error
	Retry  int64
}

// Connect creates a client connection to the Isilon cluster.
func (c *IsilonCluster) Connect() error {
	con, err := isiclient.NewIsilonClient(c.Host, c.Port, c.Username, c.PasswordEnv)
	if err != nil {
		log.Warn("Unable to create connection to the Isilon cluster.")
		return err
	}
	c.Client = con
	return nil
}

// FetchName retrieves the cluster name from the identity endpoint and stores it.
func (c *IsilonCluster) FetchName() error {
	clusterName, err := isiclient.GetClusterName(c.Client)
	if err != nil {
		log.Warnf("Unable to obtain cluster name from isi config.")
		return err
	}
	c.Name = clusterName
	log.Debugf("Cluster name is %s", c.Name)
	return nil
}

// FetchNumQuotas retrieves the number of quotas on the system.
func (c *IsilonCluster) FetchNumQuotas() error {
	summary, err := isiclient.GetQuotaSummary(c.Client)
	if err != nil {
		log.Warn("Unable to update quota summary information.")
		return err
	}
	c.Quotas.Count = int64(summary.Count)
	return nil
}

// makeConstLabels builds a prometheus.Labels map for the given cluster.
func makeConstLabels(cluster IsilonCluster) prometheus.Labels {
	if cluster.Site != "" {
		return prometheus.Labels{"cluster": cluster.Name, "site": cluster.Site}
	}
	return prometheus.Labels{"cluster": cluster.Name}
}
