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
import UnsavedChangesPrompt from './shared/saving/UnsavedChangesPrompt.vue'
import CoreFailureNotice from './shared/notices/CoreFailureNotice.vue'
import CommandPalette from './shared/command/CommandPalette.vue'
import WelcomeScreen from './welcome/WelcomeScreen.vue'
import { useWindow } from './window'
import { WORDS as words } from './shared/words'
import { WORDS as note } from './note/words'

/** What the notes still unwritten are put in: the window's words and a note's. */
const unsaved = {
  going: words.going,
  keep: note.keep,
  take: note.take,
  later: words.later,
}

const {
  carries,
  commands,
  doing,
  failure,
  going,
  held,
  layout,
  listed,
  log,
  notices,
  palette,
  places,
  shut,
  tabIcon,
  titled,
  where,
} = useWindow()
</script>

<template>
  <main>
    <CoreFailureNotice :failure="failure" />

    <WorkspaceLayout
      v-model="layout"
      class="below"
      :tabs="held.tabs.value"
      @close="shut"
      @show="held.shown"
    >
      <template #icon="{ id }">
        <component :is="tabIcon(id)" v-if="tabIcon(id)" class="tab-icon" />
      </template>

      <template #tab="{ id }">
        <component
          :is="held.heldIn(id)!.kind.draws"
          v-if="held.heldIn(id)"
          :state="held.heldIn(id)!.state"
        />

        <div v-else />
      </template>

      <template #silence>
        <WelcomeScreen
          :listed="listed"
          :tabs="held.tabs.value"
          :commands="commands"
          :search="palette"
          :doing="doing"
          :where="where"
          :carries="carries"
        />
      </template>
    </WorkspaceLayout>

    <Notices
      :notices="notices"
      :name="words.working"
      :put-away="words.putAway"
      :more="words.more"
      @gone="log.forget"
    />

    <UnsavedChangesPrompt
      :conflicts="going.conflicts.value"
      :called="titled"
      :words="unsaved"
    />

    <CommandPalette
      :commands="commands"
      :search="palette"
      :doing="doing"
      :where="where"
      :places="places"
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
