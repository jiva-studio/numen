<script setup lang="ts">
/**
 * The flashcards window: the vaults of the installation to choose from, the tab the plugins of its
 * target draw on the vault chosen, or why there is none, and the notices they show.
 */
import type { Component } from 'vue'
import '@numen/ui/styles.css'
import '@numen/shell/styles.css'
import { Notices } from '@numen/shell'
import { WORDS } from '@/shared/words'
import VaultChooser from './VaultChooser.vue'
import { useShell, type ShellDeps } from './useShell'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  /** What the shell is given in place of the window's own services. */
  deps?: ShellDeps
}>()

/* --------------------------------- State ---------------------------------- */
const { front, tab, openTab, vaults, plugins, notices, mountView } = useShell(props.deps)
</script>

<template>
  <link v-if="plugins.style.value" rel="stylesheet" :href="plugins.style.value" />
  <main class="window">
    <component
      :is="tab.view.registration.component as Component"
      v-if="tab"
      :key="tab.id"
      :state="tab.state"
    />

    <VaultChooser
      v-else-if="front"
      :vaults="vaults.list.value"
      :is-listing="vaults.isListing.value"
      :mount-view="mountView"
      @open="openTab"
    />

    <div v-else-if="!plugins.isLoading.value" class="window__empty">
      <p>{{ WORDS.empty }}</p>
      <p v-for="failure in plugins.failures.value" :key="failure">{{ failure }}</p>
    </div>
  </main>

  <!-- What the window has to say, in the corner every window says it in. -->
  <Notices :notices="notices.notices.value" @dismiss="notices.dismiss" />
</template>

<style scoped>
.window {
  display: flex;
  flex-direction: column;
  block-size: 100vh;
  padding: var(--numen-inset-wide);
  gap: var(--numen-inset);
}
</style>
