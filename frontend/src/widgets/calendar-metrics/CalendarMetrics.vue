<script setup lang="ts">
import { onMounted, ref } from "vue";
import type { CalendarMetric } from "../../entities/calendar/types";
import { fetchCalendarMetrics, updateMetricWeight } from "../../features/calendar/api";

const loading = ref(false);
const error = ref("");

const from = ref("");
const to = ref("");

const metrics = ref<CalendarMetric[]>([]);
const days = ref<string[]>([]);

const formattedDayHeader = (day: string) => {
  // day is YYYY-MM-DD
  const d = new Date(day + "T00:00:00Z");
  const weekday = d.toLocaleDateString(undefined, { weekday: "short" });
  const dayNum = d.toLocaleDateString(undefined, { day: "2-digit" });
  return `${weekday} ${dayNum}`;
};

function formatCell(cell: { number?: number; bool?: boolean } | null | undefined): string {
  if (!cell) return "-";
  if (typeof cell.number === "number") return cell.number.toString();
  if (typeof cell.bool === "boolean") return cell.bool ? "true" : "false";
  return "-";
}

function formatTarget(target: { number?: number; bool?: boolean } | undefined): string {
  if (!target) return "-";
  if (typeof target.number === "number") return target.number.toString();
  if (typeof target.bool === "boolean") return target.bool ? "true" : "false";
  return "-";
}

function formatTargetWithOperator(
  operator: "eq" | "gt" | "lt",
  target: { number?: number; bool?: boolean } | undefined
): string {
  const t = formatTarget(target);
  if (t === "-") return "-";
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

function metricTypeClass(t: CalendarMetric["metricType"]) {
  switch (t) {
    case "positive":
      return "metric-type-positive";
    case "negative":
      return "metric-type-negative";
    default:
      return "metric-type-neutral";
  }
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

function valueHighlightClass(m: CalendarMetric, cell: { number?: number; bool?: boolean } | null | undefined): string {
  if (!cell) return "";
  const match = evaluateMatch(m, cell);
  if (match) {
    switch (m.metricType) {
      case "positive":
        return "cell-good-positive";
      case "negative":
        return "cell-good-negative";
      default:
        return "cell-good-neutral";
    }
  }

  // NOTE:
  // `targetOperator` + `targetValue` decide whether the cell is a match.
  // `metricType` only affects coloring (positive/negative/neutral).
  switch (m.metricType) {
    case "positive":
      return "cell-bad-positive";
    case "negative":
      return "cell-bad-negative";
    default:
      return "cell-bad-neutral";
  }
}

const load = async () => {
  loading.value = true;
  error.value = "";

  try {
    const resp = await fetchCalendarMetrics({ from: from.value, to: to.value });
    metrics.value = resp.metrics;
    days.value = resp.days;
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Unknown error";
  } finally {
    loading.value = false;
  }
};

function setDefaultRange() {
  const now = new Date();
  const toDate = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate()));
  const fromDate = new Date(toDate);
  fromDate.setUTCDate(fromDate.getUTCDate() - 13);

  const toStr = toDate.toISOString().slice(0, 10);
  const fromStr = fromDate.toISOString().slice(0, 10);

  from.value = fromStr;
  to.value = toStr;
}

async function onWeightChange(metric: CalendarMetric) {
  try {
    await updateMetricWeight({ metricKey: metric.key, weight: metric.weight });
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Failed to update weight";
    await load();
  }
}

onMounted(() => {
  setDefaultRange();
  load();
});
</script>

<template>
  <section class="calendar-block calendar-layout">
    <header class="header">
      <h1>Calendar Metrics</h1>
      <div class="range-controls">
        <label class="range-field">
          <span>From</span>
          <input type="date" v-model="from" :disabled="loading" />
        </label>
        <label class="range-field">
          <span>To</span>
          <input type="date" v-model="to" :disabled="loading" />
        </label>
        <button class="refresh-button" :disabled="loading" @click="load">
          {{ loading ? "Loading..." : "Load" }}
        </button>
      </div>
    </header>

    <p v-if="error" class="error">{{ error }}</p>

    <section class="calendar-card" v-if="metrics.length > 0">
      <div class="calendar-scroll">
        <table class="calendar-table">
          <thead>
            <tr>
              <th class="metric-col">Metric</th>
              <th>Type</th>
              <th class="target-col">Target</th>
              <th class="weight-col">Weight</th>
              <th v-for="d in days" :key="d" class="day-col">
                {{ formattedDayHeader(d) }}
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="m in metrics" :key="m.key">
              <td class="metric-name">{{ m.title }}</td>
              <td :class="metricTypeClass(m.metricType)" class="type-badge">
                {{ m.metricType }}
              </td>
              <td class="target-cell">{{ formatTargetWithOperator(m.targetOperator, m.targetValue) }}</td>
              <td class="weight-cell">
                <input
                  class="weight-input"
                  type="number"
                  step="0.1"
                  min="0"
                  max="1000"
                  v-model.number="m.weight"
                  :disabled="loading"
                  @change="onWeightChange(m)"
                />
              </td>
              <td v-for="d in days" :key="d" class="value-cell">
                <span :class="valueHighlightClass(m, m.valuesByDate[d])">
                  {{ formatCell(m.valuesByDate[d]) }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </section>
</template>

