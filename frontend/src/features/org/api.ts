import type { OrgStructureResponse } from "../../entities/org/types";
import { requestJSON } from "../../shared/api/http";

const API_BASE = import.meta.env.VITE_API_BASE_URL || "";

export function fetchOrgStructure(): Promise<OrgStructureResponse> {
  return requestJSON<OrgStructureResponse>(`${API_BASE}/api/v1/org-structure`);
}
