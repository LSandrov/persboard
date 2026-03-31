<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import HomePage from "./pages/HomePage.vue";
import TeamsPage from "./pages/TeamsPage.vue";
import PeoplePage from "./pages/PeoplePage.vue";

const root = document.documentElement;
const currentHash = ref(window.location.hash || "#/");

function onPointerMove(event: MouseEvent) {
  const x = (event.clientX / window.innerWidth) * 100;
  const y = (event.clientY / window.innerHeight) * 100;
  root.style.setProperty("--parallax-x", `${x}%`);
  root.style.setProperty("--parallax-y", `${y}%`);
}

function onHashChange() {
  currentHash.value = window.location.hash || "#/";
}

const currentView = computed(() => {
  if (currentHash.value === "#/teams") return TeamsPage;
  if (currentHash.value === "#/people") return PeoplePage;
  return HomePage;
});

onMounted(() => {
  window.addEventListener("mousemove", onPointerMove, { passive: true });
  window.addEventListener("hashchange", onHashChange);
  onHashChange();
});

onBeforeUnmount(() => {
  window.removeEventListener("mousemove", onPointerMove);
  window.removeEventListener("hashchange", onHashChange);
});
</script>

<template>
  <div class="tech-bg" aria-hidden="true"></div>
  <nav class="app-nav">
    <a href="#/" :class="{ 'nav-link-active': currentHash === '#/' }">Главная</a>
    <a href="#/teams" :class="{ 'nav-link-active': currentHash === '#/teams' }">Команды</a>
    <a href="#/people" :class="{ 'nav-link-active': currentHash === '#/people' }">Участники</a>
  </nav>
  <component :is="currentView" />
</template>
