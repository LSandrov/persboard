import type { CalendarMetricsResponse, UpdateMetricWeightRequest } from "../../entities/calendar/types";
import { requestJSON } from "../../shared/api/http";

export async function fetchCalendarMetrics(params: { from: string; to: string }): Promise<CalendarMetricsResponse> {
  const q = new URLSearchParams();
  q.set("from", params.from);
  q.set("to", params.to);
  return requestJSON<CalendarMetricsResponse>(`/api/v1/calendar/metrics?${q.toString()}`);
}

export async function updateMetricWeight(req: UpdateMetricWeightRequest): Promise<void> {
  await fetch("/api/v1/calendar/metric-weights", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req)
  }).then(async (r) => {
    if (!r.ok) {
      const text = await r.text().catch(() => "");
      throw new Error(`Weight update failed (${r.status}): ${text || r.statusText}`);
    }
  });
}

