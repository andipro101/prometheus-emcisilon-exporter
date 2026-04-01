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
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"gopkg.in/alecthomas/kingpin.v2"
)

// Defined top level common namespace that all metrics use.
const (
	defaultEnabled  = true
	defaultDisabled = false
	namespace       = "isilon"
)

// statsEngineCallDuration and statsEngineCallFailure are shared across all
// cluster instances (cluster name is a variable label rather than a const label).
var (
	statsEngineCallDuration = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "stats_engine", "call_duration_seconds"),
		"Duration in seconds a call to the stats engine takes.",
		[]string{"stat_key", "cluster"}, nil,
	)
	statsEngineCallFailure = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "stats_engine", "call_success"),
		"0 = Successful, 1 = Failure.  Represent the successful call or failure to the stats engine.",
		[]string{"stat_key", "cluster"}, nil,
	)

	factories      = make(map[string]func(IsilonCluster) (Collector, error))
	collectorState = make(map[string]*bool)
)

func registerCollector(collector string, isDefaultEnabled bool, factory func(IsilonCluster) (Collector, error)) {
	var helpDefaultState string
	if isDefaultEnabled {
		helpDefaultState = "enabled"
	} else {
		helpDefaultState = "disabled"
	}

	flagName := fmt.Sprintf("collector.%s", collector)
	flagHelp := fmt.Sprintf("Enable the %s collector (default: %s).", collector, helpDefaultState)
	defaultValue := fmt.Sprintf("%v", isDefaultEnabled)

	flag := kingpin.Flag(flagName, flagHelp).Default(defaultValue).Bool()
	collectorState[collector] = flag

	factories[collector] = factory
}

// isilonCollector implements the prometheus.Collector interface.
type isilonCollector struct {
	Collectors           map[string]Collector
	cluster              IsilonCluster
	scrapeDurationDesc   *prometheus.Desc
	scrapeSuccessDesc    *prometheus.Desc
	exporterDurationDesc *prometheus.Desc
}

// NewIsilonCollector creates a new isilonCollector for the given cluster.
func NewIsilonCollector(cluster IsilonCluster, filters ...string) (*isilonCollector, error) {
	log.Debugf("Creating connection to the cluster endpoint %s", cluster.Host)
	if err := cluster.Connect(); err != nil {
		return nil, fmt.Errorf("unable to connect to the isilon cluster %s: %s", cluster.Host, err)
	}

	log.Debug("Getting isi config cluster name from identity endpoint.")
	if err := cluster.FetchName(); err != nil {
		return nil, fmt.Errorf("unable to get the cluster config name from the identity endpoint: %s", err)
	}

	if cluster.QuotaOnly {
		log.Debug("Setting up collector to only collect quota info.")
		if err := cluster.FetchNumQuotas(); err != nil {
			return nil, fmt.Errorf("unable to get count of quotas from the system: %s", err)
		}
	}

	constLabels := makeConstLabels(cluster)

	ic := &isilonCollector{
		cluster: cluster,
		scrapeDurationDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "scrape", "collector_duration_seconds"),
			"isilon_exporter: Duration of a collector scrape,",
			[]string{"collector"}, constLabels,
		),
		scrapeSuccessDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "scrape", "collector_success"),
			"isilon_exporter: Whether a collector succeeded.",
			[]string{"collector"}, constLabels,
		),
		exporterDurationDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "exporter", "duration_seconds"),
			"Duration in second of the entire exporter run.",
			nil, constLabels,
		),
	}

	// If quota_only, disable all collectors except quota.
	if cluster.QuotaOnly {
		var disabled = false
		var enabled = true
		for key := range collectorState {
			if key == "quota" {
				collectorState[key] = &enabled
			} else {
				collectorState[key] = &disabled
			}
		}
	}

	f := make(map[string]bool)
	for _, filter := range filters {
		enabled, exist := collectorState[filter]
		if !exist {
			return nil, fmt.Errorf("missing collector: %s", filter)
		}
		if !*enabled {
			return nil, fmt.Errorf("disabled collector: %s", filter)
		}
		f[filter] = true
	}

	collectors := make(map[string]Collector)
	for key, enabled := range collectorState {
		if *enabled {
			c, err := factories[key](cluster)
			if err != nil {
				return nil, err
			}
			if len(f) == 0 || f[key] {
				collectors[key] = c
			}
		}
	}
	ic.Collectors = collectors
	return ic, nil
}

// Describe implements the prometheus.Collector interface.
func (n isilonCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- n.scrapeDurationDesc
	ch <- n.scrapeSuccessDesc
	ch <- n.exporterDurationDesc
	ch <- statsEngineCallDuration
	ch <- statsEngineCallFailure
}

// Collect implements the prometheus.Collector interface.
func (n isilonCollector) Collect(ch chan<- prometheus.Metric) {
	begin := time.Now()
	wg := sync.WaitGroup{}
	wg.Add(len(n.Collectors))
	for name, c := range n.Collectors {
		go func(name string, c Collector) {
			n.execute(name, c, ch)
			wg.Done()
		}(name, c)
	}
	wg.Wait()
	duration := time.Since(begin)
	log.Debugf("Exporter finished after %fs", duration.Seconds())
	ch <- prometheus.MustNewConstMetric(n.exporterDurationDesc, prometheus.GaugeValue, duration.Seconds())
}

func (n isilonCollector) execute(name string, c Collector, ch chan<- prometheus.Metric) {
	begin := time.Now()
	err := c.Update(ch)
	duration := time.Since(begin)
	var success float64

	if err != nil {
		log.Errorf("ERROR: %s collector failed after %fs: %s", name, duration.Seconds(), err)
		success = 0
	} else {
		log.Debugf("OK: %s collector succeeded after %fs.", name, duration.Seconds())
		success = 1
	}
	ch <- prometheus.MustNewConstMetric(n.scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds(), name)
	ch <- prometheus.MustNewConstMetric(n.scrapeSuccessDesc, prometheus.GaugeValue, success, name)
}

// Collector is the interface a collector has to implement.
type Collector interface {
	// Get new metrics and expose them via prometheus registry.
	Update(ch chan<- prometheus.Metric) error
}
