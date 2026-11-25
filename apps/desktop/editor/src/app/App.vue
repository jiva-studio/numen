<script setup lang="ts">
/**
 * The window: one vault, and tabs to divide the screen between.
 *
 * The window's structure is made by `createWindow` and handed in. This file holds what the user
 * sees of it, and the binding between the two.
 */
import { useTemplateRef } from 'vue'
import { SlotViewHost } from '@numen/plugin-loader'
import { LoadingIndicator, provideDialogHost } from '@numen/ui'
import '@numen/ui/styles.css'
import '@numen/workbench/styles.css'
import { UnsavedChangesPrompt } from '@/features/file-conflict'
import AppContent from './AppContent.vue'
import CoreFailureNotice from './CoreFailureNotice.vue'
import { CommandPalette } from '@/features/command-palette'
import { PluginDialogs } from '@/features/plugin-host'
import type { WindowParts } from './window/createWindow'
import { WORDS as words } from '@/shared/ui/words'
import { WINDOW_WORDS } from './window/words'
import { WORDS as note } from '@/entities/note'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  parts: WindowParts
}>()

/* --------------------------------- State ---------------------------------- */
/** What the notes still unwritten are put in: the window's words and a note's. */
const unsaved = {
  going: words.going,
  keep: note.keep,
  take: note.take,
  later: words.later,
}

const { tabs, vault, getTarget, palettes, views, bootstrap } = props.parts
const { tabOpeners } = tabs
const { isHighlighted } = tabs.linkSession
const { commands, search } = palettes.palette
const { runChoice } = palettes.palette.runs
const { commandDeps, picker, dialogs } = palettes
const { pluginHost, titleBarView, statusBarView } = views
const { fileFlush, getTitle } = vault.editing
const { failure } = vault.window.health
const { isStarted } = bootstrap

/** Dialogs are drawn into the content between the title bar and the status bar, and cover only it. */
const content = useTemplateRef<HTMLElement>('content')
provideDialogHost(content)
</script>

<template>
  <!-- The window is drawn once the plugins of the window slots have started or the wait for them has
       passed, so the layout does not move when the bar is drawn. -->
  <LoadingIndicator v-if="!isStarted" class="starting" :label="WINDOW_WORDS.appName" />

  <main v-else :class="{ 'is-highlighted': isHighlighted }">
    <SlotViewHost :view="titleBarView" :styles="pluginHost.slotViews.styles" />

    <CoreFailureNotice :failure="failure" />

    <div ref="content" class="content">
      <AppContent :parts="parts" />
    </div>

    <SlotViewHost :view="statusBarView" :styles="pluginHost.slotViews.styles" />

    <UnsavedChangesPrompt
      :conflicts="fileFlush.conflicts.value"
      :get-note-title="getTitle"
      :words="unsaved"
    />

    <CommandPalette
      :commands="commands"
      :search="search"
      :doing="commandDeps"
      :get-target="getTarget"
      :places="{ findFileIcon: tabOpeners.findFileIcon }"
      :run-choice="runChoice"
      :picker="picker"
    />

    <PluginDialogs :entries="dialogs.entries.value" @fail="dialogs.close" />
  </main>
</template>

<style scoped>
main {
  display: flex;
  flex-direction: column;
  height: 100vh;
}

/* Another window offers to move a tab into this one. */
main.is-highlighted::after {
  --ring-width: 0.1875rem;
  content: '';
  position: fixed;
  inset: 0;
  z-index: var(--numen-lift-menu);
  border: var(--ring-width) solid var(--numen-accent);
  border-radius: var(--numen-window-radius);
  pointer-events: none;
}

.starting {
  height: 100vh;
}

.content {
  position: relative;
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
}
</style>
