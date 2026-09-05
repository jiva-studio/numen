<script setup lang="ts">
/**
 * The window: one vault, and tabs to divide the screen between.
 *
 * What the window is made of is `window.ts`. What is here is what a person
 * sees of it, and the binding between the two.
 */
import { Notices, Workspace } from '@numen/ui'
import '@numen/ui/styles.css'
import './app.css'
import Leaving from './Leaving.vue'
import Failure from './Failure.vue'
import Palette from './Palette.vue'
import Welcoming from './Welcoming.vue'
import { useWindow } from './window'
import { WORDS as words } from './words'

const {
  carries,
  commands,
  doing,
  failure,
  going,
  held,
  layout,
  listed,
  notices,
  palette,
  places,
  shut,
  tabIcon,
  tell,
  titled,
  where,
} = useWindow()
</script>

<template>
  <main>
    <Failure :failure="failure" />

    <Workspace
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
          :held="held.heldIn(id)!.held"
        />

        <div v-else />
      </template>

      <template #silence>
        <Welcoming
          :listed="listed"
          :tabs="held.tabs.value"
          :commands="commands"
          :search="palette"
          :doing="doing"
          :where="where"
          :carries="carries"
        />
      </template>
    </Workspace>

    <Notices
      :notices="notices"
      :name="words.working"
      :put-away="words.putAway"
      :more="words.more"
      @gone="tell.forget"
    />

    <Leaving :questions="going.questions.value" :called="titled" />

    <Palette
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
