import type { DashboardResponse } from "../../entities/dashboard/types";
import { requestJSON } from "../../shared/api/http";

const API_BASE = import.meta.env.VITE_API_BASE_URL || "";

export function fetchDashboardMetrics(): Promise<DashboardResponse> {
  return requestJSON<DashboardResponse>(`${API_BASE}/api/v1/dashboard/metrics`);
}
