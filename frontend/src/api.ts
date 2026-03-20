export type { DashboardMetric, DashboardResponse } from "./entities/dashboard/types";
export type { Person, Team, OrgStructureResponse, PeopleStats } from "./entities/org/types";

export { fetchDashboardMetrics } from "./features/dashboard/api";
export { fetchOrgStructure } from "./features/org/api";
export { fetchPeopleStats } from "./features/people/api";
