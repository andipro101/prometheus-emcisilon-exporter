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
	"net/http"
	_ "net/http/pprof"
	"sort"
	"time"

	"github.com/adobe/prometheus-emcisilon-exporter/collector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
)

// clusterMap is populated at startup and maps host -> ClusterConfig.
var clusterMap map[string]ClusterConfig

// Registers the isilon_exporter as a prometheus collector
func init() {
	version.Version = "2.0.0"
	version.BuildDate = fmt.Sprintf("%v", time.Now())
	version.BuildUser = "panike"
	prometheus.MustRegister(version.NewCollector("prometheus_emcisilon_exporter"))
}

// handler serves /metrics for a single target cluster.
func handler(w http.ResponseWriter, r *http.Request) {
	filters := r.URL.Query()["collect[]"]
	log.Debugln("collect query:", filters)

	target := r.URL.Query().Get("target")

	var cfg ClusterConfig
	if target == "" {
		// No target: if only one cluster is configured, use it.
		if len(clusterMap) == 1 {
			for _, c := range clusterMap {
				cfg = c
			}
		} else {
			msg := "missing 'target' query parameter (required when more than one cluster is configured)"
			log.Warn(msg)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(msg))
			return
		}
	} else {
		var ok bool
		cfg, ok = clusterMap[target]
		if !ok {
			msg := fmt.Sprintf("unknown target %q — not found in config", target)
			log.Warn(msg)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(msg))
			return
		}
	}

	cluster := collector.IsilonCluster{
		Host:        cfg.Host,
		Port:        cfg.Port,
		Username:    cfg.Username,
		PasswordEnv: cfg.PasswordEnv,
		Site:        cfg.Site,
		QuotaOnly:   cfg.QuotaOnly,
		Quotas: collector.Quotas{
			Retry: cfg.QuotaRetry,
		},
	}

	nc, err := collector.NewIsilonCollector(cluster, filters...)
	if err != nil {
		log.Warnf("Could not create exporter: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Could not create exporter: %s", err)))
		return
	}

	registry := prometheus.NewRegistry()
	err = registry.Register(nc)
	if err != nil {
		log.Errorf("Could not register collector: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Could not register collector: %s", err)))
		return
	}

	gatherers := prometheus.Gatherers{
		prometheus.DefaultGatherer,
		registry,
	}
	// Delegate http serving to Prometheus client library, which will call collector.Collect.
	h := promhttp.InstrumentMetricHandler(
		registry,
		promhttp.HandlerFor(gatherers,
			promhttp.HandlerOpts{
				ErrorLog:      log.NewErrorLogger(),
				ErrorHandling: promhttp.ContinueOnError,
			}),
	)
	h.ServeHTTP(w, r)
}

func main() {
	var (
		//HTTP Variables
		listenAddress = kingpin.Flag("web.listen-address", "Address on which to expose metrics and web interface.").Default(":9300").String()
		metricsPath   = kingpin.Flag("web.telemtry-path", "Path under which to expose metrics.").Default("/metrics").String()

		// Config file (preferred for multi-cluster)
		configFile = kingpin.Flag("config.file", "Path to YAML config file listing clusters. If set, individual cluster flags are ignored.").Default("").String()

		//Isilon Specific Variables (single-cluster / backward-compat mode)
		cHost     = kingpin.Flag("isilon.cluster.host", "Hostname or IP address of the isilon cluster to be scraped.").Default("").String()
		cPort     = kingpin.Flag("isilon.cluster.port", "Port to connect to the isilon cluster.").Default("8080").String()
		cUname    = kingpin.Flag("isilon.cluster.username", "Username for access the isilon API.").Default("").String()
		cPwdenv   = kingpin.Flag("isilon.cluster.password.env", "Environment variable that contains the password for the Isilon cluster user.").Default("ISILON_CLUSTER_PASSWORD").String()
		cSite     = kingpin.Flag("isilon.cluster.site", "Data Center site the cluster is located in.").Default("").String()
		quotaOnly = kingpin.Flag("quota-only", "Set exporter to only collect quota information.").Default("false").Bool()
	)

	log.AddFlags(kingpin.CommandLine)
	kingpin.Version(version.Print("prometheus-emcisilon-exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	// Build the cluster map.
	clusterMap = make(map[string]ClusterConfig)

	if *configFile != "" {
		cfg, err := loadConfig(*configFile)
		if err != nil {
			log.Fatalf("Could not load config file: %s", err)
		}
		for _, c := range cfg.Clusters {
			if c.Port == "" {
				c.Port = "8080"
			}
			if c.PasswordEnv == "" {
				c.PasswordEnv = "ISILON_CLUSTER_PASSWORD"
			}
			if c.QuotaRetry == 0 {
				c.QuotaRetry = 3
			}
			clusterMap[c.Host] = c
		}
		log.Infof("Loaded %d cluster(s) from config file %s", len(clusterMap), *configFile)
	} else {
		// Backward-compat: single cluster from flags.
		if *cUname == "" {
			log.Fatalf("No cluster username specified. Use --isilon.cluster.username or --config.file.")
		}
		if *cHost == "" {
			log.Fatalf("No cluster host specified. Use --isilon.cluster.host or --config.file.")
		}
		clusterMap[*cHost] = ClusterConfig{
			Host:        *cHost,
			Port:        *cPort,
			Username:    *cUname,
			PasswordEnv: *cPwdenv,
			Site:        *cSite,
			QuotaOnly:   *quotaOnly,
			QuotaRetry:  3,
		}
		log.Infof("Single-cluster mode: %s", *cHost)
	}

	log.Infoln("Started prometheus-emcisilon-exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	log.Infof("Configured clusters:")
	fqdns := make([]string, 0, len(clusterMap))
	for f := range clusterMap {
		fqdns = append(fqdns, f)
	}
	sort.Strings(fqdns)
	for _, f := range fqdns {
		log.Infof("  - %s", f)
	}

	http.HandleFunc(*metricsPath, handler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>Isilon Exporter</title></head>
			<body>
			<h1>Isilon Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})

	log.Infoln("Listening on", *listenAddress)
	err := http.ListenAndServe(*listenAddress, nil)
	if err != nil {
		log.Fatal(err)
	}
}
