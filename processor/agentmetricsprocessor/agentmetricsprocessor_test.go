// Copyright 2020, Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package agentmetricsprocessor

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.uber.org/zap"
)

type testCase struct {
	name                      string
	input                     pdata.Metrics
	expected                  pdata.Metrics
	prevCPUTimeValuesInput    map[string]float64
	prevCPUTimeValuesExpected map[string]float64
}

func TestAgentMetricsProcessor(t *testing.T) {
	tests := []testCase{
		{
			name:     "non-monotonic-sums-case",
			input:    generateNonMonotonicSumsInput(),
			expected: generateNonMonotonicSumsExpected(),
		},
		{
			name:     "process-resources-case",
			input:    generateProcessResourceMetricsInput(),
			expected: generateProcessResourceMetricsExpected(),
		},
		{
			name:     "read-write-split-case",
			input:    generateReadWriteMetricsInput(),
			expected: generateReadWriteMetricsExpected(),
		},
		{
			name:                      "utilization-case",
			input:                     generateUtilizationMetricsInput(),
			expected:                  generateUtilizationMetricsExpected(),
			prevCPUTimeValuesInput:    generateUtilizationPrevCPUTimeValuesInput(),
			prevCPUTimeValuesExpected: generateUtilizationPrevCPUTimeValuesExpected(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewFactory()
			tmn := &exportertest.SinkMetricsExporter{}
			rmp, err := factory.CreateMetricsProcessor(context.Background(), component.ProcessorCreateParams{Logger: zap.NewNop()}, &Config{}, tmn)
			require.NoError(t, err)

			assert.True(t, rmp.GetCapabilities().MutatesConsumedData)

			rmp.(*agentMetricsProcessor).prevCPUTimeValues = tt.prevCPUTimeValuesInput
			require.NoError(t, rmp.Start(context.Background(), componenttest.NewNopHost()))
			defer func() { assert.NoError(t, rmp.Shutdown(context.Background())) }()

			err = rmp.ConsumeMetrics(context.Background(), tt.input)
			require.NoError(t, err)

			assertEqual(t, tt.expected, tmn.AllMetrics()[0])
			assert.Equal(t, tt.prevCPUTimeValuesExpected, rmp.(*agentMetricsProcessor).prevCPUTimeValues)
		})
	}
}

// builders to generate test metrics

type resourceMetricsBuilder struct {
	rms pdata.ResourceMetricsSlice
}

func newResourceMetricsBuilder() resourceMetricsBuilder {
	return resourceMetricsBuilder{rms: pdata.NewResourceMetricsSlice()}
}

func (rmsb resourceMetricsBuilder) addResourceMetrics(resourceAttributes map[string]pdata.AttributeValue) metricsBuilder {
	rm := pdata.NewResourceMetrics()
	rm.InitEmpty()

	if resourceAttributes != nil {
		rm.Resource().Attributes().InitFromMap(resourceAttributes)
	}

	rm.InstrumentationLibraryMetrics().Resize(1)
	ilm := rm.InstrumentationLibraryMetrics().At(0)
	ilm.InitEmpty()

	rmsb.rms.Append(rm)
	return metricsBuilder{metrics: ilm.Metrics()}
}

func (rmsb resourceMetricsBuilder) Build() pdata.ResourceMetricsSlice {
	return rmsb.rms
}

type metricsBuilder struct {
	metrics pdata.MetricSlice
}

func (msb metricsBuilder) addMetric(name string, t pdata.MetricDataType, isMonotonic bool) metricBuilder {
	metric := pdata.NewMetric()
	metric.InitEmpty()
	metric.SetName(name)
	metric.SetDataType(t)

	switch t {
	case pdata.MetricDataTypeIntSum:
		sum := metric.IntSum()
		sum.InitEmpty()
		sum.SetIsMonotonic(isMonotonic)
		sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	case pdata.MetricDataTypeDoubleSum:
		sum := metric.DoubleSum()
		sum.InitEmpty()
		sum.SetIsMonotonic(isMonotonic)
		sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	case pdata.MetricDataTypeIntGauge:
		metric.IntGauge().InitEmpty()
	case pdata.MetricDataTypeDoubleGauge:
		metric.DoubleGauge().InitEmpty()
	}

	msb.metrics.Append(metric)
	return metricBuilder{metric: metric}
}

type metricBuilder struct {
	metric pdata.Metric
}

func (mb metricBuilder) addIntDataPoint(value int64, labels map[string]string) metricBuilder {
	idp := pdata.NewIntDataPoint()
	idp.InitEmpty()
	idp.LabelsMap().InitFromMap(labels)
	idp.SetValue(value)

	switch mb.metric.DataType() {
	case pdata.MetricDataTypeIntSum:
		mb.metric.IntSum().DataPoints().Append(idp)
	case pdata.MetricDataTypeIntGauge:
		mb.metric.IntGauge().DataPoints().Append(idp)
	}

	return mb
}

func (mb metricBuilder) addDoubleDataPoint(value float64, labels map[string]string) metricBuilder {
	ddp := pdata.NewDoubleDataPoint()
	ddp.InitEmpty()
	ddp.LabelsMap().InitFromMap(labels)
	ddp.SetValue(value)

	switch mb.metric.DataType() {
	case pdata.MetricDataTypeDoubleSum:
		mb.metric.DoubleSum().DataPoints().Append(ddp)
	case pdata.MetricDataTypeDoubleGauge:
		mb.metric.DoubleGauge().DataPoints().Append(ddp)
	}

	return mb
}

// assertEqual is required because Attribute & Label Maps are not sorted by default
// and we don't provide any guarantees on the order of transformed metrics
func assertEqual(t *testing.T, expected, actual pdata.Metrics) {
	rmsAct := actual.ResourceMetrics()
	rmsExp := expected.ResourceMetrics()
	require.Equal(t, rmsExp.Len(), rmsAct.Len())
	for i := 0; i < rmsAct.Len(); i++ {
		rmAct := rmsAct.At(i)
		rmExp := rmsExp.At(i)

		// assert equality of resource attributes
		assert.Equal(t, rmExp.Resource().Attributes().Sort(), rmAct.Resource().Attributes().Sort())

		// assert equality of IL metrics
		ilmsAct := rmAct.InstrumentationLibraryMetrics()
		ilmsExp := rmExp.InstrumentationLibraryMetrics()
		require.Equal(t, ilmsExp.Len(), ilmsAct.Len())
		for j := 0; j < ilmsAct.Len(); j++ {
			ilmAct := ilmsAct.At(j)
			ilmExp := ilmsExp.At(j)

			// currently expect IL to always be nil
			assert.True(t, ilmAct.InstrumentationLibrary().IsNil())
			assert.True(t, ilmExp.InstrumentationLibrary().IsNil())

			// assert equality of metrics
			metricsAct := ilmAct.Metrics()
			metricsExp := ilmExp.Metrics()
			require.Equal(t, metricsExp.Len(), metricsAct.Len())

			// build a map of expected metrics
			metricsExpMap := make(map[string]pdata.Metric, metricsExp.Len())
			for k := 0; k < metricsExp.Len(); k++ {
				metricsExpMap[metricsExp.At(k).Name()] = metricsExp.At(k)
			}

			for k := 0; k < metricsAct.Len(); k++ {
				metricAct := metricsAct.At(k)
				metricExp, ok := metricsExpMap[metricAct.Name()]
				if !ok {
					require.Fail(t, fmt.Sprintf("unexpected metric %v", metricAct.Name()))
				}

				// assert equality of descriptors
				assert.Equal(t, metricExp.Name(), metricAct.Name())
				assert.Equalf(t, metricExp.Description(), metricAct.Description(), "Metric %s", metricAct.Name())
				assert.Equalf(t, metricExp.Unit(), metricAct.Unit(), "Metric %s", metricAct.Name())
				assert.Equalf(t, metricExp.DataType(), metricAct.DataType(), "Metric %s", metricAct.Name())

				// assert equality of aggregation info & data points
				switch ty := metricAct.DataType(); ty {
				case pdata.MetricDataTypeIntSum:
					assert.Equal(t, metricAct.IntSum().AggregationTemporality(), metricExp.IntSum().AggregationTemporality(), "Metric %s", metricAct.Name())
					assert.Equal(t, metricAct.IntSum().IsMonotonic(), metricExp.IntSum().IsMonotonic(), "Metric %s", metricAct.Name())
					assertEqualIntDataPointSlice(t, metricAct.Name(), metricAct.IntSum().DataPoints(), metricExp.IntSum().DataPoints())
				case pdata.MetricDataTypeDoubleSum:
					assert.Equal(t, metricAct.DoubleSum().AggregationTemporality(), metricExp.DoubleSum().AggregationTemporality(), "Metric %s", metricAct.Name())
					assert.Equal(t, metricAct.DoubleSum().IsMonotonic(), metricExp.DoubleSum().IsMonotonic(), "Metric %s", metricAct.Name())
					assertEqualDoubleDataPointSlice(t, metricAct.Name(), metricAct.DoubleSum().DataPoints(), metricExp.DoubleSum().DataPoints())
				case pdata.MetricDataTypeIntGauge:
					assertEqualIntDataPointSlice(t, metricAct.Name(), metricAct.IntGauge().DataPoints(), metricExp.IntGauge().DataPoints())
				case pdata.MetricDataTypeDoubleGauge:
					assertEqualDoubleDataPointSlice(t, metricAct.Name(), metricAct.DoubleGauge().DataPoints(), metricExp.DoubleGauge().DataPoints())
				default:
					assert.Fail(t, "unexpected metric type", t)
				}
			}
		}
	}
}

func assertEqualIntDataPointSlice(t *testing.T, metricName string, idpsAct, idpsExp pdata.IntDataPointSlice) {
	require.Equalf(t, idpsExp.Len(), idpsAct.Len(), "Metric %s", metricName)

	// build a map of expected data points
	idpsExpMap := make(map[string]pdata.IntDataPoint, idpsExp.Len())
	for k := 0; k < idpsExp.Len(); k++ {
		idpsExpMap[labelsAsKey(idpsExp.At(k).LabelsMap())] = idpsExp.At(k)
	}

	for l := 0; l < idpsAct.Len(); l++ {
		idpAct := idpsAct.At(l)

		idpExp, ok := idpsExpMap[labelsAsKey(idpAct.LabelsMap())]
		if !ok {
			require.Failf(t, fmt.Sprintf("no data point for %s", labelsAsKey(idpAct.LabelsMap())), "Metric %s", metricName)
		}

		assert.Equalf(t, idpExp.LabelsMap().Sort(), idpAct.LabelsMap().Sort(), "Metric %s", metricName)
		assert.Equalf(t, idpExp.StartTime(), idpAct.StartTime(), "Metric %s", metricName)
		assert.Equalf(t, idpExp.Timestamp(), idpAct.Timestamp(), "Metric %s", metricName)
		assert.Equalf(t, idpExp.Value(), idpAct.Value(), "Metric %s", metricName)
	}
}

func assertEqualDoubleDataPointSlice(t *testing.T, metricName string, ddpsAct, ddpsExp pdata.DoubleDataPointSlice) {
	require.Equalf(t, ddpsExp.Len(), ddpsAct.Len(), "Metric %s", metricName)

	// build a map of expected data points
	ddpsExpMap := make(map[string]pdata.DoubleDataPoint, ddpsExp.Len())
	for k := 0; k < ddpsExp.Len(); k++ {
		ddpsExpMap[labelsAsKey(ddpsExp.At(k).LabelsMap())] = ddpsExp.At(k)
	}

	for l := 0; l < ddpsAct.Len(); l++ {
		ddpAct := ddpsAct.At(l)

		ddpExp, ok := ddpsExpMap[labelsAsKey(ddpAct.LabelsMap())]
		if !ok {
			require.Failf(t, fmt.Sprintf("no data point for %s", labelsAsKey(ddpAct.LabelsMap())), "Metric %s", metricName)
		}

		assert.Equalf(t, ddpExp.LabelsMap().Sort(), ddpAct.LabelsMap().Sort(), "Metric %s", metricName)
		assert.Equalf(t, ddpExp.StartTime(), ddpAct.StartTime(), "Metric %s", metricName)
		assert.Equalf(t, ddpExp.Timestamp(), ddpAct.Timestamp(), "Metric %s", metricName)
		assert.InDeltaf(t, ddpExp.Value(), ddpAct.Value(), 0.00000001, "Metric %s", metricName)
	}
}
