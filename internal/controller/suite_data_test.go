package controller

import (
	"encoding/json"

	"github.com/prometheus/common/model"
)

// the data in this file is used to mock Prometheus queries in controller tests

var metricjson string = `
{
	"label_topology_kubernetes_io_zone": "us-east-2a",
	"namespace": "openshift-monitoring",
	"node": "ip-10-0-146-115.us-east-2.compute.internal",
	"provider_id": "aws:///us-east-2a/i-0eb3a4cb7807fb144",
	"role": "worker",
	"pod": "hive-server-0"
}
`
var nodeallocatablecpucores string = `
[
	[
		1604685600,
		"7.5"
	]
]
`
var nodeallocatablememorybytes string = `
[
	[
		1604685600,
		"32058703872"
	]
]
`
var nodecapacitycpucores string = `
[
	[
		1604685600,
		"8"
	]
]
`
var nodecapacitymemorybytes string = `
[
	[
		1604685600,
		"33237303296"
	]
]
`
var noderole string = `
[
	[
		1604685600,
		"1"
	]
]
`
var nodelabels string = `
[
	[
		1604685600,
		"1"
	]
]`
var podlimitcpucores string = `
[
	[
		1604685600,
		"1"
	]
]
`

func asModelMatrix(metric, value string) model.Matrix {
	var m model.Metric
	var v []model.SamplePair

	if err := json.Unmarshal([]byte(metric), &m); err != nil {
		panic(err)
	}
	if err := json.Unmarshal([]byte(value), &v); err != nil {
		panic(err)
	}
	return model.Matrix{&model.SampleStream{Metric: m, Values: v}}
}
