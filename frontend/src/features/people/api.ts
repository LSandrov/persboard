import type { PeopleStats } from "../../entities/org/types";
import { requestJSON } from "../../shared/api/http";

const API_BASE = import.meta.env.VITE_API_BASE_URL || "";

export function fetchPeopleStats(): Promise<PeopleStats> {
  return requestJSON<PeopleStats>(`${API_BASE}/api/v1/people/stats`);
}

type CreatePersonPayload = {
  fullName: string;
  role: string;
  velocity: number;
  isActive: boolean;
  teamId: number;
  teamLeadId?: number;
  birthDate?: string;
  employmentDate?: string;
};

type CreatePersonResponse = {
  id: number;
  fullName: string;
};

export function createPerson(payload: CreatePersonPayload): Promise<CreatePersonResponse> {
  return requestJSON<CreatePersonResponse>(`${API_BASE}/api/v1/people`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });
}

export type UpdatePersonPayload = {
  fullName: string;
  role: string;
  velocity: number;
  isActive: boolean;
  teamId: number;
  teamLeadId?: number;
  birthDate?: string;
  employmentDate?: string;
};

export function updatePerson(personId: number, payload: UpdatePersonPayload): Promise<CreatePersonResponse> {
  return requestJSON<CreatePersonResponse>(`${API_BASE}/api/v1/people/${personId}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });
}

export async function deletePerson(personId: number): Promise<void> {
  await requestJSON<unknown>(`${API_BASE}/api/v1/people/${personId}`, {
    method: "DELETE"
  });
}
