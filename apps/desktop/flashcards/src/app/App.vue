<script setup lang="ts">
/**
 * The flashcards window: the vaults of the installation to choose from, the tab the plugins of its
 * target draw on the vault chosen, or why there is none, and the command
 * palette that sets the theme and the mode.
 */
import '@numen/ui/styles.css'
import '@numen/workbench/styles.css'
import { Palette } from '@numen/workbench'
import { SlotViewHost } from '@numen/plugin-loader'
import { LoadingIndicator } from '@numen/ui'
import { APPEARANCE_WORDS } from '@/features/appearance'
import { VaultChooser } from '@/pages/vault-chooser'
import type { WindowParts } from './window/createWindow'
import { WORDS } from './words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  parts: WindowParts
}>()

/* --------------------------------- State ---------------------------------- */
const { tabs, vaults, plugins, titleBarView, mountView, appearance } = props.parts
const { front, tab, openTab } = tabs
</script>

<template>
  <!-- The window is drawn once the plugins have started or the wait for them has passed, so the
       layout does not move when the bar is drawn. -->
  <LoadingIndicator v-if="!plugins.isStarted.value" class="starting" :label="WORDS.appName" />
  <SlotViewHost v-else :view="titleBarView" />
  <main
    v-if="plugins.isStarted.value"
    class="window"
    :class="{ 'window--below-bar': titleBarView }"
  >
    <component
      :is="tab.kind.pane"
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
    :placeholder="APPEARANCE_WORDS.palettePlaceholder"
    :name="APPEARANCE_WORDS.paletteName"
    @choose="appearance.onChoose"
    @dismiss="appearance.onDismiss"
  />
</template>

<style scoped>
.window {
  display: flex;
  flex-direction: column;
  block-size: 100vh;
  box-sizing: border-box;
  padding: var(--numen-inset-wide);
  gap: var(--numen-inset);
}

.window--below-bar {
  block-size: calc(100vh - var(--numen-title-bar-height));
}

.starting {
  block-size: 100vh;
}
</style>
