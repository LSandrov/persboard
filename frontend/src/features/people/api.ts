import type { PeopleStats } from "../../entities/org/types";
import { requestJSON } from "../../shared/api/http";

const API_BASE = import.meta.env.VITE_API_BASE_URL || "";

export function fetchPeopleStats(): Promise<PeopleStats> {
  return requestJSON<PeopleStats>(`${API_BASE}/api/v1/people/stats`);
}
