import type { OrgStructureResponse } from "../../entities/org/types";
import { requestJSON } from "../../shared/api/http";

const API_BASE = import.meta.env.VITE_API_BASE_URL || "";

export function fetchOrgStructure(): Promise<OrgStructureResponse> {
  return requestJSON<OrgStructureResponse>(`${API_BASE}/api/v1/org-structure`);
}

type CreateTeamPayload = {
  name: string;
};

type CreateTeamResponse = {
  id: number;
  name: string;
};

export function createTeam(payload: CreateTeamPayload): Promise<CreateTeamResponse> {
  return requestJSON<CreateTeamResponse>(`${API_BASE}/api/v1/teams`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });
}

export type UpdateTeamPayload = {
  name: string;
  leadId?: number;
};

export function updateTeam(teamId: number, payload: UpdateTeamPayload): Promise<CreateTeamResponse> {
  return requestJSON<CreateTeamResponse>(`${API_BASE}/api/v1/teams/${teamId}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });
}

export async function deleteTeam(teamId: number): Promise<void> {
  await requestJSON<unknown>(`${API_BASE}/api/v1/teams/${teamId}`, {
    method: "DELETE"
  });
}
