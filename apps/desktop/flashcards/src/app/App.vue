<script setup lang="ts">
/**
 * The flashcards window: the view the plugins of its target draw, or why there is none, and the
 * notices they show.
 */
import type { Component } from 'vue'
import '@numen/ui/styles.css'
import '@numen/shell/styles.css'
import { Notices } from '@numen/shell'
import { useShell } from './useShell'

/* --------------------------------- State ---------------------------------- */
const { front, style, failures, isLoading, notices } = useShell()
</script>

<template>
  <link v-if="style" rel="stylesheet" :href="style" />
  <component :is="front.registration.component as Component" v-if="front" />
  <main v-else-if="!isLoading" class="shell__empty">
    <p>No plugin of this window draws anything.</p>
    <p v-for="failure in failures" :key="failure">{{ failure }}</p>
  </main>

  <!-- What the window has to say, in the corner every window says it in. -->
  <Notices :notices="notices.notices.value" @dismiss="notices.dismiss" />
</template>

<style scoped>
.shell__empty {
  padding: var(--numen-inset-wide);
}
</style>
