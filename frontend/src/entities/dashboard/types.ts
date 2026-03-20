export type DashboardMetric = {
  key: string;
  title: string;
  value: string;
  trend: string;
};

export type DashboardResponse = {
  updatedAt: string;
  metrics: DashboardMetric[];
};
