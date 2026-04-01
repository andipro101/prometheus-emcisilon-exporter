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
	"strings"
	"time"

	"github.com/adobe/prometheus-emcisilon-exporter/isiclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

type cpuCollector struct {
	cluster   IsilonCluster
	cpuCount  *prometheus.Desc
	cpuIdle   *prometheus.Desc
	cpuUser   *prometheus.Desc
	cpuSys    *prometheus.Desc
	load1min  *prometheus.Desc
	load5min  *prometheus.Desc
	load15min *prometheus.Desc
}

func init() {
	registerCollector("cpu", defaultEnabled, NewCPUCollector)
}

// NewCPUCollector returns a new Collector exposing node cpu statistics.
func NewCPUCollector(cluster IsilonCluster) (Collector, error) {
	constLabels := makeConstLabels(cluster)
	return &cpuCollector{
		cluster: cluster,
		cpuCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, nodeCollectorSubsystem, "cpu_count"),
			"Count of number of cpu a node contains.",
			[]string{"node"}, constLabels,
		),
		cpuIdle: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, nodeCollectorSubsystem, "cpu_idle_avg"),
			"Current cpu idle percentage for the node.",
			[]string{"node"}, constLabels,
		),
		cpuUser: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, nodeCollectorSubsystem, "cpu_user_avg"),
			"Current cpu busy percentage for user mode represented in 0.0-1.0.",
			[]string{"node"}, constLabels,
		),
		cpuSys: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, nodeCollectorSubsystem, "cpu_sys_avg"),
			"Current cpu busy percentage for sys mode represented in 0.0-1.0.",
			[]string{"node"}, constLabels,
		),
		load1min: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, nodeCollectorSubsystem, "load_1min"),
			"Current 1min node load.",
			[]string{"node"}, constLabels,
		),
		load5min: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, nodeCollectorSubsystem, "load_5min"),
			"Current 5min node load.",
			[]string{"node"}, constLabels,
		),
		load15min: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, nodeCollectorSubsystem, "load_15min"),
			"Current 15min node load.",
			[]string{"node"}, constLabels,
		),
	}, nil
}

func (c *cpuCollector) Update(ch chan<- prometheus.Metric) error {
	var errCount int64
	keyMap := make(map[*prometheus.Desc]string)

	keyMap[c.cpuCount] = "node.cpu.count"
	keyMap[c.cpuIdle] = "node.cpu.idle.avg"
	keyMap[c.cpuUser] = "node.cpu.user.avg"
	keyMap[c.cpuSys] = "node.cpu.sys.avg"
	keyMap[c.load1min] = "node.load.1min"
	keyMap[c.load5min] = "node.load.5min"
	keyMap[c.load15min] = "node.load.15min"

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
				var val float64
				node := fmt.Sprintf("%v", stat.Devid)
				if strings.Contains(statKey, "cpu") {
					if !(strings.Contains(statKey, "count")) {
						val = stat.Value / 10
					}
				}
				if strings.Contains(statKey, "load") {
					val = stat.Value / 100
				}
				ch <- prometheus.MustNewConstMetric(promStat, prometheus.GaugeValue, val, node)
			}
		}
	}

	if errCount != 0 {
		err := fmt.Errorf("There where %v errors", errCount)
		return err
	}
	return nil
}
