<script setup lang="ts">
/**
 * The window: one vault, and tabs to divide the screen between.
 *
 * What the window is made of is `window.ts`. What is here is what a person
 * sees of it, and the binding between the two.
 */
import { Notices, WorkspaceLayout } from '@numen/ui'
import '@numen/ui/styles.css'
import './app.css'
import { UnsavedChangesPrompt } from '@/features/file-conflict'
import CoreFailureNotice from '@/shared/notices/CoreFailureNotice.vue'
import { CommandPalette } from '@/features/command-palette'
import { WelcomeScreen } from '@/pages/welcome'
import { useWindow } from './useWindow'
import { WORDS as words } from '@/shared/words'
import { WORDS as note } from '@/entities/note'

// --- Props & Emits ---

// --- State ---
/** What the notes still unwritten are put in: the window's words and a note's. */
const unsaved = {
  going: words.going,
  keep: note.keep,
  take: note.take,
  later: words.later,
}

const {
  runCommand,
  commands,
  commandDeps,
  failure,
  fileFlush,
  held,
  layout,
  listed,
  log,
  notices,
  palette,
  destinations,
  closeTab,
  tabIcon,
  getTitle,
  getTarget,
} = useWindow()

// --- Handlers ---
function onCloseTab(id: string) {
  closeTab(id)
}

function onShowTab(id: string) {
  held.onTabShown(id)
}

function dismissNotice(id: string) {
  log.dismiss(id)
}

// --- Helpers ---
</script>

<template>
  <main>
    <CoreFailureNotice :failure="failure" />

    <WorkspaceLayout
      v-model="layout"
      class="below"
      :tabs="held.tabs.value"
      @close="onCloseTab"
      @show="onShowTab"
    >
      <template #icon="{ id }">
        <component :is="tabIcon(id)" v-if="tabIcon(id)" class="tab-icon" />
      </template>

      <template #tab="{ id }">
        <component
          :is="held.getTab(id)!.kind.pane"
          v-if="held.getTab(id)"
          :state="held.getTab(id)!.state"
        />

        <div v-else />
      </template>

      <template #silence>
        <WelcomeScreen
          :listed="listed"
          :tabs="held.tabs.value"
          :commands="commands"
          :search="palette"
          :doing="commandDeps"
          :get-target="getTarget"
          :run-command="runCommand"
        />
      </template>
    </WorkspaceLayout>

    <Notices
      :notices="notices"
      :name="words.working"
      :dismiss="words.dismiss"
      :more="words.more"
      @dismiss="dismissNotice"
    />

    <UnsavedChangesPrompt
      :conflicts="fileFlush.conflicts.value"
      :called="getTitle"
      :words="unsaved"
    />

    <CommandPalette
      :commands="commands"
      :search="palette"
      :doing="commandDeps"
      :get-target="getTarget"
      :places="destinations"
    />
  </main>
</template>

<style scoped>
main {
  display: flex;
  flex-direction: column;
  height: 100vh;
}

.below {
  flex: 1;
  min-height: 0;
}

/* Lucide draws on a 24 grid, and the stroke is given in those units. */
.tab-icon {
  inline-size: 100%;
  block-size: 100%;
  stroke-width: 1.75;
  opacity: 0.75;
}
</style>
