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
	"encoding/json"
	"fmt"
	"time"

	"github.com/adobe/prometheus-emcisilon-exporter/isiclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

type clusterProtoCollector struct {
	cluster                     IsilonCluster
	clusterProtocolInMax        *prometheus.Desc
	clusterProtocolInMin        *prometheus.Desc
	clusterProtocolInRate       *prometheus.Desc
	clusterProtocolOpCount      *prometheus.Desc
	clusterProtocolOpRate       *prometheus.Desc
	clusterProtocolOutMax       *prometheus.Desc
	clusterProtocolOutMin       *prometheus.Desc
	clusterProtocolOutRate      *prometheus.Desc
	clusterProtocolTimeAvg      *prometheus.Desc
	clusterProtocolTimeMax      *prometheus.Desc
	clusterProtocolTimeMin      *prometheus.Desc
	clusterProtocolTotalInMax   *prometheus.Desc
	clusterProtocolTotalInMin   *prometheus.Desc
	clusterProtocolTotalInRate  *prometheus.Desc
	clusterProtocolTotalOpCount *prometheus.Desc
	clusterProtocolTotalOpRate  *prometheus.Desc
	clusterProtocolTotalOutMax  *prometheus.Desc
	clusterProtocolTotalOutMin  *prometheus.Desc
	clusterProtocolTotalOutRate *prometheus.Desc
	clusterProtocolTotalTimeAvg *prometheus.Desc
	clusterProtocolTotalTimeMax *prometheus.Desc
	clusterProtocolTotalTimeMin *prometheus.Desc
}

func init() {
	registerCollector("cluster_protocol", defaultEnabled, NewClusterProtoCollector)
	if !protosUpdated {
		GetProtos()
	}
}

// NewClusterProtoCollector returns a new Collector exposing cluster protocol statistics.
func NewClusterProtoCollector(cluster IsilonCluster) (Collector, error) {
	constLabels := makeConstLabels(cluster)
	return &clusterProtoCollector{
		cluster: cluster,
		clusterProtocolInMax: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, clusterCollectorSubsystem, "protostats_in_max"),
			"Cluster protocol operation in max.",
			[]string{"proto", "op"}, constLabels,
		),
		clusterProtocolInMin: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, clusterCollectorSubsystem, "protostats_in_min"),
			"Cluster protocol operation in min.",
			[]string{"proto", "op"}, constLabels,
		),
		clusterProtocolInRate: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, clusterCollectorSubsystem, "protostats_in_rate"),
			"Cluster protocol operation in rate.",
			[]string{"proto", "op"}, constLabels,
		),
		clusterProtocolOpCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, clusterCollectorSubsystem, "protostats_op_count"),
			"Cluster protocol operation count.",
			[]string{"proto", "op"}, constLabels,
		),
		clusterProtocolOpRate: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, clusterCollectorSubsystem, "protostats_op_rate"),
			"Cluster protocol operation rate.",
			[]string{"proto", "op"}, constLabels,
		),
		clusterProtocolOutMax: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, clusterCollectorSubsystem, "protostats_out_max"),
			"Cluster protocol operation out max.",
			[]string{"proto", "op"}, constLabels,
		),
		clusterProtocolOutMin: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, clusterCollectorSubsystem, "protostats_out_min"),
			"Cluster protocol operation out min.",
			[]string{"proto", "op"}, constLabels,
		),
		clusterProtocolOutRate: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, clusterCollectorSubsystem, "protostats_out_rate"),
			"Cluster protocol operation out rate.",
			[]string{"proto", "op"}, constLabels,
		),
		clusterProtocolTimeAvg: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, clusterCollectorSubsystem, "protostats_time_avg"),
			"Cluster protocol operation time average.",
			[]string{"proto", "op"}, constLabels,
		),
		clusterProtocolTimeMax: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, clusterCollectorSubsystem, "protostats_time_max"),
			"Cluster protocol operation time max.",
			[]string{"proto", "op"}, constLabels,
		),
		clusterProtocolTimeMin: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, clusterCollectorSubsystem, "protostats_time_min"),
			"Total cluster protocol operation in rate.",
			[]string{"proto", "op"}, constLabels,
		),
		clusterProtocolTotalInMax: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, clusterCollectorSubsystem, "protostats_in_max_total"),
			"Total cluster protocol operation in max.",
			[]string{"proto"}, constLabels,
		),
		clusterProtocolTotalInMin: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, clusterCollectorSubsystem, "protostats_in_min_total"),
			"Total cluster protocol operation in min.",
			[]string{"proto"}, constLabels,
		),
		clusterProtocolTotalInRate: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, clusterCollectorSubsystem, "protostats_in_rate_total"),
			"Total cluster protocol operation in rate.",
			[]string{"proto"}, constLabels,
		),
		clusterProtocolTotalOpCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, clusterCollectorSubsystem, "protostats_op_count_total"),
			"Total cluster protocol operation count.",
			[]string{"proto"}, constLabels,
		),
		clusterProtocolTotalOpRate: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, clusterCollectorSubsystem, "protostats_op_rate_total"),
			"Total cluster protocol operation rate.",
			[]string{"proto"}, constLabels,
		),
		clusterProtocolTotalOutMax: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, clusterCollectorSubsystem, "protostats_out_max_total"),
			"Total cluster protocol operation out max.",
			[]string{"proto"}, constLabels,
		),
		clusterProtocolTotalOutMin: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, clusterCollectorSubsystem, "protostats_out_min_total"),
			"Total cluster protocol operation out min.",
			[]string{"proto"}, constLabels,
		),
		clusterProtocolTotalOutRate: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, clusterCollectorSubsystem, "protostats_out_rate_total"),
			"Total cluster protocol operation out rate.",
			[]string{"proto"}, constLabels,
		),
		clusterProtocolTotalTimeAvg: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, clusterCollectorSubsystem, "protostats_time_avg_total"),
			"Total cluster protocol operation time average.",
			[]string{"proto"}, constLabels,
		),
		clusterProtocolTotalTimeMax: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, clusterCollectorSubsystem, "protostats_time_max_total"),
			"Total cluster protocol operation time max.",
			[]string{"proto"}, constLabels,
		),
		clusterProtocolTotalTimeMin: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, clusterCollectorSubsystem, "protostats_time_min_total"),
			"Total cluster protocol operation in rate.",
			[]string{"proto"}, constLabels,
		),
	}, nil
}

func (c *clusterProtoCollector) Update(ch chan<- prometheus.Metric) error {
	//Set client stats for nfs and smb to ungathered
	nfsClientstatGathered = false
	smbClientstatsGathered = false

	var err error
	var errCount int64
	//Attempt to update over the list of all protocol
	for proto, state := range protocolState {
		// Only execute if state is true
		if *state {
			err = c.updateProtoOpStats(ch, proto)
			if err != nil {
				log.Warnf("Unabled to collect protocol operation stats for %s", proto)
				errCount++
			}
			err = c.updateProtoStats(ch, proto)
			if err != nil {
				log.Warnf("Unable to collect protocol stats for %s", proto)
				errCount++
			}
		}
	}

	if errCount != 0 {
		err = fmt.Errorf("There where %v errors", errCount)
		return err
	}
	return nil
}

func (c *clusterProtoCollector) updateProtoOpStats(ch chan<- prometheus.Metric, protocol string) error {
	key := fmt.Sprintf("cluster.protostats.%v", protocol)
	begin := time.Now()
	resp, err := isiclient.GetProtoStat(c.cluster.Client, key)
	duration := time.Since(begin)
	ch <- prometheus.MustNewConstMetric(statsEngineCallDuration, prometheus.GaugeValue, duration.Seconds(), key, c.cluster.Name)
	if err != nil {
		log.Warnf("Unable to collect protocol stats for protocol %s.", protocol)
		ch <- prometheus.MustNewConstMetric(statsEngineCallFailure, prometheus.GaugeValue, 1, key, c.cluster.Name)
		return err
	}
	ch <- prometheus.MustNewConstMetric(statsEngineCallFailure, prometheus.GaugeValue, 0, key, c.cluster.Name)
	for _, stat := range resp.Stats {
		if stat.Value != nil {
			values := stat.Value.([]interface{})
			if len(values) > 0 {
				for _, value := range values {
					var protoStat isiclient.IsiProtoStatOp
					j, jerr := json.Marshal(value)
					if err != nil {
						err = jerr
						log.Warnf("Could not marshal back into json: %v", err)
					}
					err = json.Unmarshal(j, &protoStat)
					if err != nil {
						log.Warnf("Could not unmarshl into stuct: %v", err)
					}
					ch <- prometheus.MustNewConstMetric(c.clusterProtocolInMax, prometheus.GaugeValue, protoStat.InMax, protocol, protoStat.OpName)
					ch <- prometheus.MustNewConstMetric(c.clusterProtocolInMin, prometheus.GaugeValue, protoStat.InMin, protocol, protoStat.OpName)
					ch <- prometheus.MustNewConstMetric(c.clusterProtocolInRate, prometheus.GaugeValue, protoStat.InRate, protocol, protoStat.OpName)
					ch <- prometheus.MustNewConstMetric(c.clusterProtocolOpCount, prometheus.GaugeValue, protoStat.OpCount, protocol, protoStat.OpName)
					ch <- prometheus.MustNewConstMetric(c.clusterProtocolOpRate, prometheus.GaugeValue, protoStat.OpRate, protocol, protoStat.OpName)
					ch <- prometheus.MustNewConstMetric(c.clusterProtocolOutMax, prometheus.GaugeValue, protoStat.OutMax, protocol, protoStat.OpName)
					ch <- prometheus.MustNewConstMetric(c.clusterProtocolOutMin, prometheus.GaugeValue, protoStat.OutMin, protocol, protoStat.OpName)
					ch <- prometheus.MustNewConstMetric(c.clusterProtocolOutRate, prometheus.GaugeValue, protoStat.OutRate, protocol, protoStat.OpName)
					ch <- prometheus.MustNewConstMetric(c.clusterProtocolTimeAvg, prometheus.GaugeValue, protoStat.TimeAvg, protocol, protoStat.OpName)
					ch <- prometheus.MustNewConstMetric(c.clusterProtocolTimeMax, prometheus.GaugeValue, protoStat.TimeMax, protocol, protoStat.OpName)
					ch <- prometheus.MustNewConstMetric(c.clusterProtocolTimeMin, prometheus.GaugeValue, protoStat.TimeMin, protocol, protoStat.OpName)
				}
			}
		}

	}

	return err
}

func (c *clusterProtoCollector) updateProtoStats(ch chan<- prometheus.Metric, protocol string) error {
	key := fmt.Sprintf("cluster.protostats.%s.total", protocol)
	begin := time.Now()
	resp, err := isiclient.GetProtoStat(c.cluster.Client, key)
	duration := time.Since(begin)
	ch <- prometheus.MustNewConstMetric(statsEngineCallDuration, prometheus.GaugeValue, duration.Seconds(), key, c.cluster.Name)
	if err != nil {
		log.Warnf("Unable to collect cluster protocol stats for protocol %s.", protocol)
		ch <- prometheus.MustNewConstMetric(statsEngineCallFailure, prometheus.GaugeValue, 1, key, c.cluster.Name)
		return err
	}
	ch <- prometheus.MustNewConstMetric(statsEngineCallFailure, prometheus.GaugeValue, 0, key, c.cluster.Name)

	for _, stat := range resp.Stats {
		if stat.Value != nil {
			values := stat.Value.([]interface{})

			if len(values) > 0 {
				for _, value := range values {
					var protoStat isiclient.IsiProtoStatTotal
					j, jerr := json.Marshal(value)
					if jerr != nil {
						err = jerr
						log.Warnf("Could not marshal back into json: %v", err)
					}
					err = json.Unmarshal(j, &protoStat)
					if err != nil {
						log.Warnf("Could not unmarshl into stuct: %v", err)
					}
					ch <- prometheus.MustNewConstMetric(c.clusterProtocolTotalInMax, prometheus.GaugeValue, protoStat.InMax, protocol)
					ch <- prometheus.MustNewConstMetric(c.clusterProtocolTotalInMin, prometheus.GaugeValue, protoStat.InMin, protocol)
					ch <- prometheus.MustNewConstMetric(c.clusterProtocolTotalInRate, prometheus.GaugeValue, protoStat.InRate, protocol)
					ch <- prometheus.MustNewConstMetric(c.clusterProtocolTotalOpCount, prometheus.GaugeValue, protoStat.OpCount, protocol)
					ch <- prometheus.MustNewConstMetric(c.clusterProtocolTotalOpRate, prometheus.GaugeValue, protoStat.OpRate, protocol)
					ch <- prometheus.MustNewConstMetric(c.clusterProtocolTotalOutMax, prometheus.GaugeValue, protoStat.OutMax, protocol)
					ch <- prometheus.MustNewConstMetric(c.clusterProtocolTotalOutMin, prometheus.GaugeValue, protoStat.OutMin, protocol)
					ch <- prometheus.MustNewConstMetric(c.clusterProtocolTotalOutRate, prometheus.GaugeValue, protoStat.OutRate, protocol)
					ch <- prometheus.MustNewConstMetric(c.clusterProtocolTotalTimeAvg, prometheus.GaugeValue, protoStat.TimeAvg, protocol)
					ch <- prometheus.MustNewConstMetric(c.clusterProtocolTotalTimeMax, prometheus.GaugeValue, protoStat.TimeMax, protocol)
					ch <- prometheus.MustNewConstMetric(c.clusterProtocolTotalTimeMin, prometheus.GaugeValue, protoStat.TimeMin, protocol)
				}
			}
		}
	}

	return err
}
