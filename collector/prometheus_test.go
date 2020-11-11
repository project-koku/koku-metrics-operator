/*


Copyright 2020 Red Hat, Inc.

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package collector

import (
	"context"
	"reflect"
	"testing"
	"time"

	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type mappedMockPromResult map[string]*mockPromResult
type mockPromResult struct {
	value    model.Value
	warnings promv1.Warnings
	err      error
}
type mockPrometheusConnection struct {
	mappedResults mappedMockPromResult
	singleResult  *mockPromResult
	t             *testing.T
}

func (m mockPrometheusConnection) QueryRange(ctx context.Context, query string, r promv1.Range) (model.Value, promv1.Warnings, error) {
	var res *mockPromResult
	var ok bool
	if m.mappedResults != nil {
		res, ok = m.mappedResults[query]
		if !ok {
			m.t.Fatalf("Could not find test result!")
		}
	} else if m.singleResult != nil {
		res = m.singleResult
	} else {
		m.t.Fatalf("Could not find test result!")
	}
	return res.value, res.warnings, res.err
}

func (m mockPrometheusConnection) Query(ctx context.Context, query string, ts time.Time) (model.Value, promv1.Warnings, error) {
	res := m.singleResult
	return res.value, res.warnings, res.err
}

func TestPerformMatrixQuery(t *testing.T) {
	col := PromCollector{
		TimeSeries: &promv1.Range{},
		Log:        zap.New(),
	}
	performMatrixQueryTests := []struct {
		name        string
		query       string
		queryResult *mockPromResult
		want        model.Matrix
		err         error
	}{
		{
			name:        "return incorrect type (model.Scalar)",
			query:       "fake-query",
			queryResult: &mockPromResult{value: &model.Scalar{}},
			want:        nil,
			err:         errTest,
		},
		{
			name:        "return incorrect type (model.Vector)",
			query:       "fake-query",
			queryResult: &mockPromResult{value: &model.Vector{}},
			want:        nil,
			err:         errTest,
		},
		{
			name:        "return incorrect type (model.String)",
			query:       "fake-query",
			queryResult: &mockPromResult{value: &model.String{}},
			want:        nil,
			err:         errTest,
		},
	}
	for _, tt := range performMatrixQueryTests {
		t.Run(tt.name, func(t *testing.T) {
			col.PromConn = mockPrometheusConnection{
				singleResult: tt.queryResult,
				t:            t,
			}
			got, err := col.performMatrixQuery(tt.query)
			if err != nil && tt.err == nil {
				t.Errorf("%s got unexpected error: %v", tt.name, err)
			}
			if tt.err != nil && err == nil {
				t.Errorf("%s got `%v` error, wanted error", tt.name, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%s got %s want %s", tt.name, got, tt.want)
			}
		})
	}
}

func TestGetQueryResults(t *testing.T) {
	col := PromCollector{
		TimeSeries: &promv1.Range{},
		Log:        zap.New(),
	}
	getQueryResultsErrorsTests := []struct {
		name          string
		queries       *querys
		queriesResult mappedMockPromResult
		wantedResult  mappedResults
		wantedError   error
	}{
		{
			name: "get query results no errors",
			queries: &querys{
				query{
					Name:        "usage-cpu-cores",
					QueryString: "query1",
					MetricKey:   staticFields{"id": "id"},
					QueryValue: &saveQueryValue{
						ValName:         "usage-cpu-cores",
						Method:          "max",
						Factor:          maxFactor,
						TransformedName: "usage-cpu-core-seconds",
					},
					RowKey: "id",
				},
				query{
					Name:        "capacity-cpu-cores",
					QueryString: "query2",
					MetricKey:   staticFields{"id": "id"},
					QueryValue: &saveQueryValue{
						ValName:         "capacity-cpu-cores",
						Method:          "max",
						Factor:          maxFactor,
						TransformedName: "capacity-cpu-core-seconds",
					},
					RowKey: "id",
				},
				query{
					Name:           "labels",
					QueryString:    "query3",
					MetricKeyRegex: regexFields{"labels": "label_*"},
					RowKey:         "id",
				},
			},
			queriesResult: mappedMockPromResult{
				"query1": &mockPromResult{
					value: model.Matrix{
						{
							Metric: model.Metric{
								"id":           "1",
								"random-field": "42",
							},
							Values: []model.SamplePair{
								{Timestamp: 1604339340, Value: 2},
								{Timestamp: 1604339400, Value: 2},
								{Timestamp: 1604339460, Value: 2},
							},
						}},
					warnings: nil,
					err:      nil,
				},
				"query2": &mockPromResult{
					value: model.Matrix{
						{
							Metric: model.Metric{"id": "1"},
							Values: []model.SamplePair{
								{Timestamp: 1604339340, Value: 4},
								{Timestamp: 1604339400, Value: 4},
								{Timestamp: 1604339460, Value: 4},
							},
						},
					},
					warnings: nil,
					err:      nil,
				},
				"query3": &mockPromResult{
					value: model.Matrix{
						{
							Metric: model.Metric{
								"id":            "1",
								"label_arch":    "amd64",
								"label_io_zone": "us-east-2c",
							},
							Values: []model.SamplePair{
								{Timestamp: 1604339340, Value: 1},
								{Timestamp: 1604339400, Value: 1},
								{Timestamp: 1604339460, Value: 1},
							},
						},
					},
					warnings: nil,
					err:      nil,
				},
			},
			wantedResult: mappedResults{
				"1": {
					"id":                        "1",
					"usage-cpu-cores":           "2.000000",
					"usage-cpu-core-seconds":    "360.000000",
					"capacity-cpu-cores":        "4.000000",
					"capacity-cpu-core-seconds": "720.000000",
					"labels":                    "label_arch:amd64|label_io_zone:us-east-2c",
				},
			},
			wantedError: nil,
		},
	}
	for _, tt := range getQueryResultsErrorsTests {
		t.Run(tt.name, func(t *testing.T) {
			col.PromConn = mockPrometheusConnection{
				mappedResults: tt.queriesResult,
				t:             t,
			}
			got := mappedResults{}
			err := col.getQueryResults(tt.queries, &got)
			if err != nil && tt.wantedError != nil {
				t.Errorf("got unexpected error: %v", err)
			}
			if tt.wantedError != nil && err == nil {
				t.Errorf("%s got: nil error, want: error", tt.name)
			}
			if !reflect.DeepEqual(got, tt.wantedResult) {
				t.Errorf("getQueryResults got:\n\t%s\n  want:\n\t%s", got, tt.wantedResult)
			}
		})
	}
}

func TestGetQueryResultsError(t *testing.T) {
	col := PromCollector{
		TimeSeries: &promv1.Range{},
		Log:        zap.New(),
	}
	getQueryResultsErrorsTests := []struct {
		name         string
		query        query
		queryResult  *mockPromResult
		wantedResult mappedResults
		wantedError  error
	}{
		{
			name:  "warnings with no error",
			query: query{QueryString: "fake-query-1", RowKey: "node"},
			queryResult: &mockPromResult{
				value:    model.Matrix{},
				warnings: promv1.Warnings{"This is a warning."},
				err:      nil,
			},
			wantedResult: mappedResults{},
			wantedError:  nil,
		},
		{
			name:  "error with no warnings",
			query: query{QueryString: "fake-query-2", RowKey: "node"},
			queryResult: &mockPromResult{
				value:    model.Matrix{},
				warnings: nil,
				err:      errTest,
			},
			wantedResult: mappedResults{},
			wantedError:  errTest,
		},
		{
			name:  "error with warnings",
			query: query{QueryString: "fake-query-3", RowKey: "node"},
			queryResult: &mockPromResult{
				value:    model.Matrix{},
				warnings: promv1.Warnings{"This is another warning."},
				err:      errTest,
			},
			wantedResult: mappedResults{},
			wantedError:  errTest,
		},
	}
	for _, tt := range getQueryResultsErrorsTests {
		t.Run(tt.name, func(t *testing.T) {
			col.PromConn = mockPrometheusConnection{
				singleResult: tt.queryResult,
				t:            t,
			}
			got := mappedResults{}
			err := col.getQueryResults(&querys{tt.query}, &got)
			if err != nil && tt.wantedError == nil {
				t.Errorf("%s got unexpected error: %v", tt.name, err)
			}
			if got != nil {
				eq := reflect.DeepEqual(got, tt.wantedResult)
				if !eq {
					t.Errorf("%s got: %s want: %s", tt.name, got, tt.wantedResult)
				}
			}
			if tt.wantedError != nil && err == nil {
				t.Errorf("%s got: nil error, want: error", tt.name)
			}
		})
	}
}

func TestTestPrometheusConnection(t *testing.T) {
	col := PromCollector{
		TimeSeries: &promv1.Range{},
		Log:        zap.New(),
	}
	testPrometheusConnectionTests := []struct {
		name        string
		wait        time.Duration
		queryResult *mockPromResult
		wantedError error
	}{
		{
			name:        "test query success",
			queryResult: &mockPromResult{err: nil},
			wantedError: nil,
		},
		{
			name:        "test query error",
			queryResult: &mockPromResult{err: errTest},
			wantedError: errTest,
		},
	}
	for _, tt := range testPrometheusConnectionTests {
		t.Run(tt.name, func(t *testing.T) {
			col.PromConn = mockPrometheusConnection{
				singleResult: tt.queryResult,
				t:            t,
			}
			err := col.testPrometheusConnection()
			if err != nil && tt.wantedError == nil {
				t.Errorf("%s got unexpected error: %v", tt.name, err)
			}
			if tt.wantedError != nil && err == nil {
				t.Errorf("%s got: %v error, want: error", tt.name, err)
			}
		})
	}
}
