<script setup lang="ts">
import { useDashboardOverview } from "./useDashboardOverview";
import CalendarMetrics from "../calendar-metrics/CalendarMetrics.vue";
import DepartmentStatus from "./DepartmentStatus.vue";

const {
  loading,
  error,
  metrics,
  teams,
  peopleStats,
  orgSummary,
  formattedUpdatedAt,
  loadDashboard
} = useDashboardOverview();
</script>

<template>
  <main class="layout">
    <header class="header">
      <h1>Team Leads Dashboard</h1>
      <button class="refresh-button" :disabled="loading" @click="loadDashboard">
        {{ loading ? "Loading..." : "Refresh" }}
      </button>
    </header>

    <p class="updated-at">Updated: {{ formattedUpdatedAt }}</p>

    <p v-if="error" class="error">{{ error }}</p>

    <section class="metrics-grid">
      <article v-for="item in metrics" :key="item.key" class="metric-card">
        <h2>{{ item.title }}</h2>
        <p class="metric-value">{{ item.value }}</p>
        <p class="metric-trend">{{ item.trend }}</p>
      </article>
    </section>

    <section class="details-card">
      <h2>Кратко по структуре</h2>
      <p>Команд: {{ orgSummary.totalTeams }}</p>
      <p>Участников: {{ orgSummary.totalMembers }}</p>
    </section>

    <DepartmentStatus />

    <CalendarMetrics />

    <section v-if="peopleStats" class="details-card">
      <h2>People Statistics</h2>
      <p>Total people: {{ peopleStats.totalPeople }}</p>
      <p>Active people: {{ peopleStats.activePeople }}</p>
      <p>Average velocity: {{ peopleStats.averageVelocity.toFixed(1) }}</p>
    </section>

    <section class="details-card">
      <h2>Org Structure</h2>
      <div v-if="teams.length === 0">No teams yet</div>
      <div v-else class="teams-grid">
        <article v-for="team in teams" :key="team.id" class="team-card">
          <h3>{{ team.name }}</h3>
          <p>Members: {{ team.members.length }}</p>
          <ul>
            <li v-for="member in team.members" :key="member.id">
              {{ member.fullName }} - {{ member.role }} ({{ member.velocity }})
              <span v-if="!member.isActive"> [inactive]</span>
            </li>
          </ul>
        </article>
      </div>
    </section>
  </main>
</template>
