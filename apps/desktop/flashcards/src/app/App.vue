<script setup lang="ts">
/**
 * The flashcards window: the vaults of the installation to choose from, the tab the plugins of its
 * target draw on the vault chosen, or why there is none, and the command
 * palette that sets the theme and the mode.
 */
import type { Component } from 'vue'
import '@numen/ui/styles.css'
import '@numen/shell/styles.css'
import { Palette, TitleBar } from '@numen/shell'
import { WORDS } from '@/shared/words'
import VaultChooser from './VaultChooser.vue'
import { useShell, type ShellDeps } from './useShell'
import { useWindowFrame } from './useWindowFrame'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  /** What the shell is given in place of the window's own services. */
  deps?: ShellDeps
}>()

/* --------------------------------- State ---------------------------------- */
const { front, tab, openTab, vaults, plugins, mountView, appearance } = useShell(props.deps)
const frame = useWindowFrame()
</script>

<template>
  <link v-if="plugins.style.value" rel="stylesheet" :href="plugins.style.value" />
  <TitleBar
    :app-name="WORDS.appName"
    :vault-name="WORDS.windowName"
    :controls="frame.controls.value"
    :is-maximised="frame.isMaximised.value"
    @minimise="frame.minimise"
    @toggle-maximise="frame.toggleMaximise"
    @close="frame.close"
  />
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

  <!-- The appearance commands: the themes and the modes of a color pair. -->
  <Palette
    v-model="appearance.query.value"
    :groups="appearance.groups.value"
    :is-open="appearance.isOpen.value"
    :placeholder="WORDS.palettePlaceholder"
    :name="WORDS.paletteName"
    @choose="appearance.onChoose"
    @dismiss="appearance.onDismiss"
  />
</template>

<style scoped>
.window {
  display: flex;
  flex-direction: column;
  block-size: calc(100vh - 2rem);
  box-sizing: border-box;
  padding: var(--numen-inset-wide);
  gap: var(--numen-inset);
}
</style>
