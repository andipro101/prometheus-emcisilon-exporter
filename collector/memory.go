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
	"fmt"
	"time"

	"github.com/adobe/prometheus-emcisilon-exporter/isiclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

type memoryCollector struct {
	cluster     IsilonCluster
	memoryUsed  *prometheus.Desc
	memoryFree  *prometheus.Desc
	memoryCache *prometheus.Desc
}

func init() {
	registerCollector("memory", defaultEnabled, NewMemoryCollector)
}

// NewMemoryCollector returns a new Collector exposing node memory statistics.
func NewMemoryCollector(cluster IsilonCluster) (Collector, error) {
	constLabels := makeConstLabels(cluster)
	return &memoryCollector{
		cluster: cluster,
		memoryUsed: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, nodeCollectorSubsystem, "memory_used"),
			"RAM memory currently in use in bytes.",
			[]string{"node"}, constLabels,
		),
		memoryFree: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, nodeCollectorSubsystem, "memory_free"),
			"RAM memory currently free in bytes.",
			[]string{"node"}, constLabels,
		),
		memoryCache: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, nodeCollectorSubsystem, "memory_cache"),
			"RAM memory currently used for cache in bytes.",
			[]string{"node"}, constLabels,
		),
	}, nil
}

func (c *memoryCollector) Update(ch chan<- prometheus.Metric) error {
	var errCount int64
	keyMap := make(map[*prometheus.Desc]string)

	keyMap[c.memoryUsed] = "node.memory.used"
	keyMap[c.memoryFree] = "node.memory.free"
	keyMap[c.memoryCache] = "node.memory.cache"

	for promStat, statKey := range keyMap {
		begin := time.Now()
		resp, err := isiclient.QueryStatsEngineSingleVal(c.cluster.Client, statKey)
		duration := time.Since(begin)
		ch <- prometheus.MustNewConstMetric(statsEngineCallDuration, prometheus.GaugeValue, duration.Seconds(), statKey, c.cluster.Name)
		if err != nil {
			log.Warnf("Error attempting to query stats engine with key %s: %s", statKey, err)
			ch <- prometheus.MustNewConstMetric(statsEngineCallFailure, prometheus.GaugeValue, 1, statKey, c.cluster.Name)
			errCount++
		} else {
			ch <- prometheus.MustNewConstMetric(statsEngineCallFailure, prometheus.GaugeValue, 0, statKey, c.cluster.Name)
			for _, stat := range resp.Stats {
				node := fmt.Sprintf("%v", stat.Devid)
				ch <- prometheus.MustNewConstMetric(promStat, prometheus.GaugeValue, stat.Value, node)
			}
		}
	}
	if errCount != 0 {
		return fmt.Errorf("There where %d errors", errCount)
	}
	return nil
}
