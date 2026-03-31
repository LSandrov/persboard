<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import type { Team } from "../entities/org/types";
import { createTeam, deleteTeam, fetchOrgStructure, updateTeam } from "../features/org/api";

const loading = ref(false);
const submitting = ref(false);
const error = ref("");
const success = ref("");
const teamName = ref("");
const teams = ref<Team[]>([]);
const editingTeamId = ref<number | null>(null);
const editingTeamName = ref("");

const isSubmitDisabled = computed(() => submitting.value || teamName.value.trim().length === 0);

async function loadTeams() {
  loading.value = true;
  error.value = "";
  try {
    const data = await fetchOrgStructure();
    teams.value = data.teams;
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Unknown error";
  } finally {
    loading.value = false;
  }
}

async function submitTeam() {
  if (isSubmitDisabled.value) return;

  submitting.value = true;
  error.value = "";
  success.value = "";
  try {
    const created = await createTeam({ name: teamName.value.trim() });
    success.value = `Команда "${created.name}" добавлена`;
    teamName.value = "";
    await loadTeams();
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Unknown error";
  } finally {
    submitting.value = false;
  }
}

function startEditTeam(team: Team) {
  editingTeamId.value = team.id;
  editingTeamName.value = team.name;
}

function cancelEditTeam() {
  editingTeamId.value = null;
  editingTeamName.value = "";
}

async function saveTeam(teamId: number) {
  const name = editingTeamName.value.trim();
  if (!name) return;

  submitting.value = true;
  error.value = "";
  success.value = "";
  try {
    await updateTeam(teamId, { name });
    success.value = "Команда обновлена";
    cancelEditTeam();
    await loadTeams();
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Unknown error";
  } finally {
    submitting.value = false;
  }
}

async function removeTeam(team: Team) {
  const isConfirmed = window.confirm(`Удалить команду "${team.name}"?`);
  if (!isConfirmed) return;

  submitting.value = true;
  error.value = "";
  success.value = "";
  try {
    await deleteTeam(team.id);
    success.value = "Команда удалена";
    if (editingTeamId.value === team.id) {
      cancelEditTeam();
    }
    await loadTeams();
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Unknown error";
  } finally {
    submitting.value = false;
  }
}

onMounted(loadTeams);
</script>

<template>
  <main class="layout">
    <header class="header">
      <h1>Управление командами</h1>
      <button class="refresh-button" :disabled="loading || submitting" @click="loadTeams">
        {{ loading ? "Загрузка..." : "Обновить" }}
      </button>
    </header>

    <section class="details-card">
      <h2>Добавить команду</h2>
      <form class="form-grid" @submit.prevent="submitTeam">
        <label class="range-field">
          Название
          <input v-model="teamName" type="text" maxlength="100" placeholder="Например, Platform Team" />
        </label>
        <button class="refresh-button" :disabled="isSubmitDisabled" type="submit">
          {{ submitting ? "Сохраняем..." : "Добавить" }}
        </button>
      </form>
      <p v-if="success" class="success">{{ success }}</p>
      <p v-if="error" class="error">{{ error }}</p>
    </section>

    <section class="details-card">
      <h2>Список команд</h2>
      <p v-if="teams.length === 0">Команд пока нет</p>
      <div v-else class="teams-grid">
        <article v-for="team in teams" :key="team.id" class="team-card">
          <template v-if="editingTeamId === team.id">
            <input v-model="editingTeamName" class="inline-input" type="text" maxlength="120" />
          </template>
          <h3 v-else>{{ team.name }}</h3>
          <p>Участников: {{ team.members.length }}</p>
          <div class="row-actions">
            <template v-if="editingTeamId === team.id">
              <button class="refresh-button" :disabled="submitting || !editingTeamName.trim()" @click="saveTeam(team.id)">
                Сохранить
              </button>
              <button class="secondary-button" :disabled="submitting" @click="cancelEditTeam">
                Отмена
              </button>
            </template>
            <template v-else>
              <button class="secondary-button" :disabled="submitting" @click="startEditTeam(team)">Редактировать</button>
              <button class="danger-button" :disabled="submitting" @click="removeTeam(team)">Удалить</button>
            </template>
          </div>
        </article>
      </div>
    </section>
  </main>
</template>
