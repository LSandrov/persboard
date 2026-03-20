<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import type { CalendarMetric, CalendarMetricsResponse } from "../../entities/calendar/types";
import { fetchCalendarMetrics } from "../../features/calendar/api";

const loading = ref(false);
const error = ref("");

const selectedDate = ref<string>("");
const response = ref<CalendarMetricsResponse | null>(null);

function formatTargetWithOperator(
  operator: "eq" | "gt" | "lt",
  target: { number?: number; bool?: boolean } | undefined
): string {
  if (!target) return "-";
  if (typeof target.number === "number") {
    const t = target.number.toString();
    switch (operator) {
      case "gt":
        return `>${t}`;
      case "lt":
        return `<${t}`;
      case "eq":
      default:
        return `=${t}`;
    }
  }
  if (typeof target.bool === "boolean") {
    const t = target.bool ? "true" : "false";
    return operator === "eq" ? `=${t}` : `=${t}`;
  }
  return "-";
}

function evaluateMatch(m: CalendarMetric, cell: { number?: number; bool?: boolean } | null | undefined): boolean {
  if (!cell) return false;

  const op = m.targetOperator;
  if (typeof m.targetValue.number === "number") {
    if (typeof cell.number !== "number") return false;
    const tv = m.targetValue.number;
    const cv = cell.number;
    const eps = 1e-9;
    switch (op) {
      case "gt":
        return cv > tv;
      case "lt":
        return cv < tv;
      case "eq":
      default:
        return Math.abs(cv - tv) <= eps;
    }
  }

  if (typeof m.targetValue.bool === "boolean") {
    if (typeof cell.bool !== "boolean") return false;
    switch (op) {
      case "eq":
      default:
        return cell.bool === m.targetValue.bool;
    }
  }

  return false;
}

function formatCell(cell: { number?: number; bool?: boolean } | null | undefined): string {
  if (!cell) return "-";
  if (typeof cell.number === "number") return cell.number.toString();
  if (typeof cell.bool === "boolean") return cell.bool ? "true" : "false";
  return "-";
}

function signedContribution(m: CalendarMetric, cell: { number?: number; bool?: boolean } | null | undefined): number {
  const match = evaluateMatch(m, cell);
  // For scoring we also take `metricType` into account:
  // - positive/negative: miss target => full penalty
  // - neutral: miss target => softer penalty
  const mismatchMultiplier = m.metricType === "neutral" ? 0.5 : 1;
  return match ? m.weight : -m.weight * mismatchMultiplier;
}

const totalWeight = computed(() => {
  return response.value?.metrics.reduce((acc, m) => acc + m.weight, 0) ?? 0;
});

const signedScore = computed(() => {
  if (!response.value) return 0;
  const day = selectedDate.value || response.value.days?.[0] || "";
  return response.value.metrics.reduce((acc, m) => {
    const cell = m.valuesByDate?.[day] ?? null;
    return acc + signedContribution(m, cell);
  }, 0);
});

const scoreNormalized = computed(() => {
  const tw = totalWeight.value;
  if (tw <= 0) return 0;
  return signedScore.value / tw; // -1..+1
});

const departmentState = computed(() => {
  const s = scoreNormalized.value;
  if (s >= 0.6) return { label: "Отдел в порядке", tone: "ok" as const };
  if (s >= 0) return { label: "Отдел под контролем", tone: "warn" as const };
  return { label: "Отдел требует внимания", tone: "bad" as const };
});

const departmentStateClass = computed(() => {
  switch (departmentState.value.tone) {
    case "ok":
      return "dept-status-badge dept-status-ok";
    case "warn":
      return "dept-status-badge dept-status-warn";
    case "bad":
    default:
      return "dept-status-badge dept-status-bad";
  }
});

async function load() {
  if (!selectedDate.value) return;
  loading.value = true;
  error.value = "";

  try {
    const resp = await fetchCalendarMetrics({ from: selectedDate.value, to: selectedDate.value });
    response.value = resp;
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Unknown error";
    response.value = null;
  } finally {
    loading.value = false;
  }
}

function setDefaultDate() {
  // Use UTC date string so it matches backend parsing.
  const now = new Date();
  selectedDate.value = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate())).toISOString().slice(0, 10);
}

onMounted(() => {
  setDefaultDate();
  load();
});
</script>

<template>
  <section class="details-card">
    <h2>Отдел: состояние по метрикам</h2>

    <div class="dept-status-controls">
      <label class="range-field">
        <span>Дата</span>
        <input type="date" v-model="selectedDate" :disabled="loading" />
      </label>
      <button class="refresh-button" :disabled="loading" @click="load">
        {{ loading ? "Loading..." : "Показать" }}
      </button>
    </div>

    <p v-if="error" class="error">{{ error }}</p>

    <div v-if="response" class="dept-status-summary">
      <div class="dept-status-row">
        <span :class="departmentStateClass">
          {{ departmentState.label }}
        </span>
        <span class="dept-score">
          Score: {{ signedScore }} / {{ totalWeight }} ({{ (scoreNormalized * 100).toFixed(0) }}%)
        </span>
      </div>

      <div class="dept-metrics-breakdown">
        <h3>Влияние метрик на {{ selectedDate }}</h3>
        <table class="dept-metrics-table">
          <thead>
            <tr>
              <th>Метрика</th>
              <th>Тип</th>
              <th>Цель</th>
              <th>Значение</th>
              <th>Вес</th>
              <th>Статус</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="m in response.metrics" :key="m.key">
              <td class="dept-metric-title">{{ m.title }}</td>
              <td>{{ m.metricType }}</td>
              <td>{{ formatTargetWithOperator(m.targetOperator, m.targetValue) }}</td>
              <td class="dept-metric-value">
                {{ formatCell(m.valuesByDate?.[selectedDate] ?? null) }}
              </td>
              <td class="dept-metric-weight">{{ m.weight }}</td>
              <td>
                <span
                  :class="{
                    'cell-good-positive': m.metricType === 'positive' && evaluateMatch(m, m.valuesByDate?.[selectedDate]),
                    'cell-good-negative': m.metricType === 'negative' && evaluateMatch(m, m.valuesByDate?.[selectedDate]),
                    'cell-good-neutral': m.metricType === 'neutral' && evaluateMatch(m, m.valuesByDate?.[selectedDate]),
                    'cell-bad-positive': m.metricType === 'positive' && !evaluateMatch(m, m.valuesByDate?.[selectedDate]),
                    'cell-bad-negative': m.metricType === 'negative' && !evaluateMatch(m, m.valuesByDate?.[selectedDate]),
                    'cell-bad-neutral': m.metricType === 'neutral' && !evaluateMatch(m, m.valuesByDate?.[selectedDate])
                  }"
                  class="dept-metric-pill"
                >
                  {{ evaluateMatch(m, m.valuesByDate?.[selectedDate]) ? "OK" : "BAD" }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </section>
</template>

<style scoped>
.dept-status-controls {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.dept-status-summary {
  margin-top: 12px;
}

.dept-status-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.dept-score {
  color: #475569;
  font-variant-numeric: tabular-nums;
}

.dept-metrics-breakdown {
  margin-top: 14px;
}

.dept-metrics-table {
  width: 100%;
  border-collapse: collapse;
}

.dept-metrics-table th,
.dept-metrics-table td {
  border-bottom: 1px solid #e2e8f0;
  padding: 8px 10px;
  text-align: left;
  white-space: nowrap;
}

.dept-metric-title {
  font-weight: 600;
}

.dept-metric-value {
  font-variant-numeric: tabular-nums;
}

.dept-metric-weight {
  font-variant-numeric: tabular-nums;
}

.dept-metric-pill {
  display: inline-block;
}
</style>

