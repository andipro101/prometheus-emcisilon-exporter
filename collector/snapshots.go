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
	"math"
	"time"

	"github.com/adobe/prometheus-emcisilon-exporter/isiclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

type snapshotsCollector struct {
	cluster                IsilonCluster
	snapshotsTotalCount    *prometheus.Desc
	snapshotsTotalSize     *prometheus.Desc
	snapshotsActiveCount   *prometheus.Desc
	snapshotsActiveSize    *prometheus.Desc
	snapshotsDeletingCount *prometheus.Desc
	snapshotsDeletingSize  *prometheus.Desc
	snapshots7DayCount     *prometheus.Desc
	snapshots15DayCount    *prometheus.Desc
	snapshots30DayCount    *prometheus.Desc
	snapshots60DayCount    *prometheus.Desc
	snapshots90DayCount    *prometheus.Desc
}

func init() {
	registerCollector("snapshots", defaultEnabled, NewSnapshotsCollector)
}

// NewSnapshotsCollector returns a new Collector exposing snapshot information.
func NewSnapshotsCollector(cluster IsilonCluster) (Collector, error) {
	constLabels := makeConstLabels(cluster)
	return &snapshotsCollector{
		cluster: cluster,
		snapshots7DayCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "snapshots", "7_day_count"),
			"Number of snapshots older than 7 days.",
			nil, constLabels,
		),
		snapshots15DayCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "snapshots", "15_day_count"),
			"Number of snapshots older than 15 days.",
			nil, constLabels,
		),
		snapshots30DayCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "snapshots", "30_day_count"),
			"Number of snapshots older than 30 days",
			nil, constLabels,
		),
		snapshots60DayCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "snapshots", "60_day_count"),
			"Number of snapshots older than 60 days.",
			nil, constLabels,
		),
		snapshots90DayCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "snapshots", "90_day_count"),
			"Number of snapshots older than 90 days.",
			nil, constLabels,
		),
		snapshotsActiveCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "snapshots", "active_count"),
			"Number of snapshots that are active on the system.",
			nil, constLabels,
		),
		snapshotsActiveSize: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "snapshots", "active_size"),
			"Size in bytes of space occupied by active snapshots.",
			nil, constLabels,
		),
		snapshotsDeletingCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "snapshots", "deleting_count"),
			"Number of snapshots that are being deleted from the system.",
			nil, constLabels,
		),
		snapshotsDeletingSize: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "snapshots", "deleting_size"),
			"Size in bytes of space occupied by snapshots being deleted. ",
			nil, constLabels,
		),
		snapshotsTotalCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "snapshots", "total_count"),
			"Total number of snapshots (both active and deleting) on a cluster.",
			nil, constLabels,
		),
		snapshotsTotalSize: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "snapshots", "total_size"),
			"Size in bytes of space occupides by all snapshots.",
			nil, constLabels,
		),
	}, nil
}

func (c *snapshotsCollector) Update(ch chan<- prometheus.Metric) error {
	err := c.updateSummary(ch)
	if err != nil {
		log.Warnf("Unabled to update snapshot summary information: %s", err)
	}

	err = c.updateDayCounts(ch)
	if err != nil {
		log.Warnf("Unable to update snapshot day thresholds: %s", err)
	}

	if err != nil {
		return err
	}
	return nil
}

func (c *snapshotsCollector) updateSummary(ch chan<- prometheus.Metric) error {
	resp, err := isiclient.GetSnapshotsSummary(c.cluster.Client)
	if err != nil {
		return err
	}
	ch <- prometheus.MustNewConstMetric(c.snapshotsActiveCount, prometheus.GaugeValue, resp.Summary.ActiveCount)
	ch <- prometheus.MustNewConstMetric(c.snapshotsActiveSize, prometheus.GaugeValue, resp.Summary.ActiveSize)
	ch <- prometheus.MustNewConstMetric(c.snapshotsDeletingCount, prometheus.GaugeValue, resp.Summary.DeletingCount)
	ch <- prometheus.MustNewConstMetric(c.snapshotsDeletingSize, prometheus.GaugeValue, resp.Summary.DeletingSize)
	ch <- prometheus.MustNewConstMetric(c.snapshotsTotalCount, prometheus.GaugeValue, resp.Summary.Count)
	ch <- prometheus.MustNewConstMetric(c.snapshotsTotalSize, prometheus.GaugeValue, resp.Summary.Size)

	return nil
}

func (c *snapshotsCollector) updateDayCounts(ch chan<- prometheus.Metric) error {
	type TimeThreshold struct {
		Name     string
		Days     int64
		Counter  int
		PromDesc *prometheus.Desc
	}

	var thresholds = []TimeThreshold{
		{"7d", 7, 0, c.snapshots7DayCount},
		{"15d", 15, 0, c.snapshots15DayCount},
		{"30d", 30, 0, c.snapshots30DayCount},
		{"60d", 60, 0, c.snapshots60DayCount},
		{"90d", 90, 0, c.snapshots90DayCount},
	}

	resp, err := isiclient.GetSnapshots(c.cluster.Client)
	if err != nil {
		return err
	}

	for _, snapshot := range resp.Snapshots {
		elapsed := time.Since(time.Unix(snapshot.Created, 0))
		days := RoundTime(elapsed.Seconds() / 86400)
		for idx := range thresholds {
			if days >= thresholds[idx].Days {
				thresholds[idx].Counter++
			} else {
				break
			}
		}
	}

	for idx := range thresholds {
		ch <- prometheus.MustNewConstMetric(thresholds[idx].PromDesc, prometheus.GaugeValue, float64(thresholds[idx].Counter))
	}
	return nil
}

// RoundTime - Well gotta deal with those floating point numbers somehow
func RoundTime(input float64) int64 {
	var result float64

	if input < 0 {
		result = math.Ceil(input - 0.5)
	} else {
		result = math.Floor(input + 0.5)
	}

	// only interested in integer, ignore fractional
	i, _ := math.Modf(result)

	return int64(i)
}
