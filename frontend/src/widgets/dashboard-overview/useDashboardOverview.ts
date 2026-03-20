import { computed, onMounted, ref } from "vue";
import type { DashboardMetric } from "../../entities/dashboard/types";
import type { PeopleStats, Team } from "../../entities/org/types";
import { fetchDashboardMetrics } from "../../features/dashboard/api";
import { fetchPeopleStats } from "../../features/people/api";
import { fetchOrgStructure } from "../../features/org/api";

export function useDashboardOverview() {
  const loading = ref(true);
  const error = ref("");
  const updatedAt = ref("");
  const metrics = ref<DashboardMetric[]>([]);
  const peopleStats = ref<PeopleStats | null>(null);
  const teams = ref<Team[]>([]);

  const formattedUpdatedAt = computed(() => {
    if (!updatedAt.value) return "-";
    return new Date(updatedAt.value).toLocaleString();
  });

  async function loadDashboard() {
    loading.value = true;
    error.value = "";

    try {
      const [dashboardData, statsData, orgData] = await Promise.all([
        fetchDashboardMetrics(),
        fetchPeopleStats(),
        fetchOrgStructure()
      ]);

      metrics.value = dashboardData.metrics;
      updatedAt.value = dashboardData.updatedAt;
      peopleStats.value = statsData;
      teams.value = orgData.teams;
    } catch (e) {
      error.value = e instanceof Error ? e.message : "Unknown error";
    } finally {
      loading.value = false;
    }
  }

  onMounted(loadDashboard);

  return {
    loading,
    error,
    metrics,
    teams,
    peopleStats,
    formattedUpdatedAt,
    loadDashboard
  };
}
