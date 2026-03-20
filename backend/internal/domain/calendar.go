package domain

import "time"

type MetricType string

const (
	MetricTypeNeutral  MetricType = "neutral"
	MetricTypePositive MetricType = "positive"
	MetricTypeNegative MetricType = "negative"
)

type TargetValue struct {
	Number *float64 `json:"number,omitempty"`
	Bool   *bool    `json:"bool,omitempty"`
}

type TargetOperator string

const (
	TargetOperatorEq TargetOperator = "eq"
	TargetOperatorGt TargetOperator = "gt"
	TargetOperatorLt TargetOperator = "lt"
)

type CalendarMetricDefinition struct {
	Key              string         `json:"key"`
	Title            string         `json:"title"`
	DefaultWeight    float64        `json:"defaultWeight"`
	MetricType       MetricType     `json:"metricType"`
	TargetValue      TargetValue    `json:"targetValue"`
	TargetOperator   TargetOperator `json:"targetOperator"`
	EazyBIReportID   int            `json:"eazybiReportId"`
	EazyBIFormat     string         `json:"eazybiFormat"`
	TimeMemberFormat string         `json:"timeMemberFormat"`
}

type MetricWeight struct {
	Key    string  `json:"key"`
	Title  string  `json:"title"`
	Weight float64 `json:"weight"`
}

type CalendarMetricCellValue struct {
	Number *float64 `json:"number,omitempty"`
	Bool   *bool    `json:"bool,omitempty"`
}

type CalendarMetric struct {
	Key            string                              `json:"key"`
	Title          string                              `json:"title"`
	Weight         float64                             `json:"weight"`
	MetricType     MetricType                          `json:"metricType"`
	TargetValue    TargetValue                         `json:"targetValue"`
	TargetOperator TargetOperator                      `json:"targetOperator"`
	ValuesByDate   map[string]*CalendarMetricCellValue `json:"valuesByDate"`
}

type CalendarMetricsResponse struct {
	From    string           `json:"from"`
	To      string           `json:"to"`
	Days    []string         `json:"days"`
	Metrics []CalendarMetric `json:"metrics"`
}

type UpdateMetricWeightInput struct {
	MetricKey string  `json:"metricKey"`
	Weight    float64 `json:"weight"`
}

func DateToYMD(t time.Time) string {
	return t.Format("2006-01-02")
}
