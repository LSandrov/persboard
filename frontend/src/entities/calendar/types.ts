export type CalendarMetric = {
  key: string;
  title: string;
  // semantic coloring:
  // - positive: "higher is better" (for numeric metrics, depending on targetOperator)
  // - negative: "lower is better" (for numeric metrics, depending on targetOperator)
  // - neutral: no preference
  metricType: "neutral" | "positive" | "negative";
  targetValue: { number?: number; bool?: boolean };
  targetOperator: "eq" | "gt" | "lt";
  weight: number;
  valuesByDate: Record<string, { number?: number; bool?: boolean } | null>;
};

export type CalendarMetricsResponse = {
  from: string;
  to: string;
  days: string[];
  metrics: CalendarMetric[];
};

export type UpdateMetricWeightRequest = {
  metricKey: string;
  weight: number;
};

