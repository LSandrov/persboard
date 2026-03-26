<script setup lang="ts">
import { onBeforeUnmount, onMounted } from "vue";
import DashboardOverview from "./widgets/dashboard-overview/DashboardOverview.vue";

const root = document.documentElement;

function onPointerMove(event: MouseEvent) {
  const x = (event.clientX / window.innerWidth) * 100;
  const y = (event.clientY / window.innerHeight) * 100;
  root.style.setProperty("--parallax-x", `${x}%`);
  root.style.setProperty("--parallax-y", `${y}%`);
}

onMounted(() => {
  window.addEventListener("mousemove", onPointerMove, { passive: true });
});

onBeforeUnmount(() => {
  window.removeEventListener("mousemove", onPointerMove);
});
</script>

<template>
  <div class="tech-bg" aria-hidden="true"></div>
  <DashboardOverview />
</template>
