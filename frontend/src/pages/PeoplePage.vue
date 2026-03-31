<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import type { Team } from "../entities/org/types";
import { fetchOrgStructure } from "../features/org/api";
import { createPerson, deletePerson, updatePerson } from "../features/people/api";

const loading = ref(false);
const submitting = ref(false);
const error = ref("");
const success = ref("");
const teams = ref<Team[]>([]);

const fullName = ref("");
const role = ref("");
const velocity = ref<number>(1);
const isActive = ref(true);
const teamId = ref<number | null>(null);
const birthDate = ref("");
const employmentDate = ref("");
const editingPersonId = ref<number | null>(null);
const editFullName = ref("");
const editRole = ref("");
const editVelocity = ref<number>(0);
const editIsActive = ref(true);
const editTeamId = ref<number | null>(null);
const editBirthDate = ref("");
const editEmploymentDate = ref("");

const selectedTeam = computed(() => teams.value.find((team) => team.id === teamId.value) ?? null);
const isSubmitDisabled = computed(() => {
  return (
    submitting.value ||
    !fullName.value.trim() ||
    !role.value.trim() ||
    teamId.value === null ||
    Number.isNaN(Number(velocity.value))
  );
});

async function loadTeams() {
  loading.value = true;
  error.value = "";
  try {
    const data = await fetchOrgStructure();
    teams.value = data.teams;
    if (teamId.value === null && teams.value.length > 0) {
      teamId.value = teams.value[0].id;
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Unknown error";
  } finally {
    loading.value = false;
  }
}

async function submitPerson() {
  if (isSubmitDisabled.value || teamId.value === null) return;

  submitting.value = true;
  error.value = "";
  success.value = "";

  try {
    const created = await createPerson({
      fullName: fullName.value.trim(),
      role: role.value.trim(),
      velocity: Number(velocity.value),
      isActive: isActive.value,
      teamId: teamId.value,
      birthDate: birthDate.value || undefined,
      employmentDate: employmentDate.value || undefined
    });
    success.value = `Участник "${created.fullName}" добавлен`;
    fullName.value = "";
    role.value = "";
    velocity.value = 1;
    isActive.value = true;
    birthDate.value = "";
    employmentDate.value = "";
    await loadTeams();
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Unknown error";
  } finally {
    submitting.value = false;
  }
}

function startEditPerson(person: Team["members"][number]) {
  editingPersonId.value = person.id;
  editFullName.value = person.fullName;
  editRole.value = person.role;
  editVelocity.value = person.velocity;
  editIsActive.value = person.isActive;
  editTeamId.value = person.teamId;
  editBirthDate.value = person.birthDate ?? "";
  editEmploymentDate.value = person.employmentDate ?? "";
}

function cancelEditPerson() {
  editingPersonId.value = null;
  editFullName.value = "";
  editRole.value = "";
  editVelocity.value = 0;
  editIsActive.value = true;
  editTeamId.value = null;
  editBirthDate.value = "";
  editEmploymentDate.value = "";
}

async function savePerson(personId: number) {
  if (editTeamId.value === null) return;

  submitting.value = true;
  error.value = "";
  success.value = "";
  try {
    await updatePerson(personId, {
      fullName: editFullName.value.trim(),
      role: editRole.value.trim(),
      velocity: Number(editVelocity.value),
      isActive: editIsActive.value,
      teamId: editTeamId.value,
      birthDate: editBirthDate.value || undefined,
      employmentDate: editEmploymentDate.value || undefined
    });
    success.value = "Участник обновлен";
    cancelEditPerson();
    await loadTeams();
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Unknown error";
  } finally {
    submitting.value = false;
  }
}

async function removePerson(person: Team["members"][number]) {
  const isConfirmed = window.confirm(`Удалить участника "${person.fullName}"?`);
  if (!isConfirmed) return;

  submitting.value = true;
  error.value = "";
  success.value = "";
  try {
    await deletePerson(person.id);
    success.value = "Участник удален";
    if (editingPersonId.value === person.id) {
      cancelEditPerson();
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
      <h1>Управление участниками</h1>
      <button class="refresh-button" :disabled="loading || submitting" @click="loadTeams">
        {{ loading ? "Загрузка..." : "Обновить" }}
      </button>
    </header>

    <section class="details-card">
      <h2>Добавить участника в команду</h2>
      <form class="form-grid" @submit.prevent="submitPerson">
        <label class="range-field">
          ФИО
          <input v-model="fullName" type="text" maxlength="120" placeholder="Например, Иван Петров" />
        </label>
        <label class="range-field">
          Роль
          <input v-model="role" type="text" maxlength="80" placeholder="Например, Backend Developer" />
        </label>
        <label class="range-field">
          Команда
          <select v-model.number="teamId">
            <option v-for="team in teams" :key="team.id" :value="team.id">
              {{ team.name }}
            </option>
          </select>
        </label>
        <label class="range-field">
          Velocity
          <input v-model.number="velocity" type="number" step="0.1" min="0" />
        </label>
        <label class="checkbox-field">
          <input v-model="isActive" type="checkbox" />
          Активный сотрудник
        </label>
        <label class="range-field">
          Дата рождения
          <input v-model="birthDate" type="date" />
        </label>
        <label class="range-field">
          Дата трудоустройства
          <input v-model="employmentDate" type="date" />
        </label>
        <button class="refresh-button" :disabled="isSubmitDisabled" type="submit">
          {{ submitting ? "Сохраняем..." : "Добавить" }}
        </button>
      </form>
      <p v-if="success" class="success">{{ success }}</p>
      <p v-if="error" class="error">{{ error }}</p>
    </section>

    <section class="details-card">
      <h2>Участники в выбранной команде</h2>
      <template v-if="selectedTeam">
        <h3>{{ selectedTeam.name }}</h3>
        <p v-if="selectedTeam.members.length === 0">В этой команде пока нет участников</p>
        <ul v-else>
          <li v-for="member in selectedTeam.members" :key="member.id">
            <template v-if="editingPersonId === member.id">
              <div class="form-grid">
                <input v-model="editFullName" class="inline-input" type="text" maxlength="120" />
                <input v-model="editRole" class="inline-input" type="text" maxlength="60" />
                <input v-model.number="editVelocity" class="inline-input" type="number" min="0" max="100" step="0.1" />
                <input v-model="editBirthDate" class="inline-input" type="date" />
                <input v-model="editEmploymentDate" class="inline-input" type="date" />
                <select v-model.number="editTeamId" class="inline-input">
                  <option v-for="team in teams" :key="team.id" :value="team.id">
                    {{ team.name }}
                  </option>
                </select>
                <label class="checkbox-field">
                  <input v-model="editIsActive" type="checkbox" />
                  Активен
                </label>
              </div>
              <div class="row-actions">
                <button
                  class="refresh-button"
                  :disabled="
                    submitting ||
                    !editFullName.trim() ||
                    !editRole.trim() ||
                    editTeamId === null ||
                    Number.isNaN(Number(editVelocity))
                  "
                  @click="savePerson(member.id)"
                >
                  Сохранить
                </button>
                <button class="secondary-button" :disabled="submitting" @click="cancelEditPerson">Отмена</button>
              </div>
            </template>
            <template v-else>
              {{ member.fullName }} - {{ member.role }} ({{ member.velocity }})
              <span v-if="!member.isActive"> [inactive]</span>
              <span v-if="member.birthDate"> | ДР: {{ member.birthDate }}</span>
              <span v-if="member.employmentDate"> | Трудоустроен: {{ member.employmentDate }}</span>
              <div class="row-actions">
                <button class="secondary-button" :disabled="submitting" @click="startEditPerson(member)">Редактировать</button>
                <button class="danger-button" :disabled="submitting" @click="removePerson(member)">Удалить</button>
              </div>
            </template>
          </li>
        </ul>
      </template>
      <p v-else>Сначала добавьте команду</p>
    </section>
  </main>
</template>
