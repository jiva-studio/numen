<script setup lang="ts">
/**
 * One preset's tile: what its goal is, how far through it the day stands, and
 * what a session on it would ask.
 */
import { Button } from '@numen/ui'

import type { Tile } from '../../lib/tiles'

defineProps<{ tile: Tile }>()

defineEmits<{
  /** Sit down to every deck this preset schedules. */
  (event: 'start'): void
}>()
</script>

<template>
  <!-- Starting a session on a preset is the same act as starting a session on a deck,
       one level up, so it is the same button. One with nothing to offer
       is not pressed, and has said above why. -->
  <Button
    variant="outline"
    class="presets__preset"
    :class="{ 'presets__preset--paused': tile.one.paused }"
    :disabled="!tile.canStart"
    @click="$emit('start')"
  >
    <span class="presets__said">
      <span class="presets__name">{{ tile.one.name }}</span>
      <span v-if="tile.goal" class="presets__goal">{{ tile.goal }}</span>
      <!-- What is wrong with the preset, where the goal it could not
           state would stand. The editor is where it is settled. -->
      <span v-if="tile.wrong" class="presets__wrong">{{ tile.wrong }}</span>
    </span>

    <!-- How far through the day it is, and under it what pressing this
         would ask or why it would ask nothing. A preset scheduling
         nothing has said so under its name, and stands here empty.

         A session is left back onto this screen with these figures moved,
         so they are said again where they are read aloud. -->
    <span v-if="!tile.one.paused" class="presets__figures" aria-live="polite">
      <span class="presets__done" :data-over="tile.isOver ? '' : undefined">{{ tile.says }}</span>
      <span v-if="tile.left" class="presets__left">{{ tile.left }}</span>
    </span>
  </Button>
</template>

<style scoped>
/* A tile is a deck of the list below one level up, so it is the same button.
   What it is painted, how it answers a hover and what it looks like with the
   keyboard on it are the button's own. It reads across, with what the day comes
   to at its right. */
.presets__preset {
  inline-size: 100%;
  block-size: auto;
  justify-content: space-between;
  padding: var(--numen-inset);
  gap: var(--numen-inset);
  text-align: start;
  white-space: normal;
}

/* The name, with the goal under it. */
.presets__said {
  display: flex;
  min-inline-size: 0;
  flex-direction: column;
  gap: 0.125rem;
}

.presets__name {
  font-weight: 600;
}

.presets__goal {
  color: var(--numen-hushed);
}

/* What is wrong with the preset, which is the one thing under the name worth
   catching the eye. */
.presets__wrong {
  color: var(--numen-caution-fg);
  font-size: var(--numen-text-1);
}

/* How far through the day it is, with what pressing it would ask under that. */
.presets__figures {
  display: flex;
  flex: none;
  flex-direction: column;
  align-items: end;
}

/* What a person looks for, so it carries the weight in the tile and stands at
   the end of the line, against the middle of the two beside it. */
.presets__done {
  font-size: var(--numen-title-size);
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

/* What the session would ask, in the small print a count is read in. */
.presets__left {
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  font-weight: 400;
  white-space: nowrap;
}

/* A day over its budget is the one thing on the tile worth catching the eye,
   and it is words, which read at the size words do. */
.presets__done[data-over] {
  color: var(--numen-caution-fg);
  font-size: var(--numen-font-size);
  text-align: end;
}

/* A preset scheduling nothing today stands with its reason and nothing else. */
.presets__preset--paused .presets__name,
.presets__preset--paused .presets__goal {
  color: var(--numen-hushed);
}
</style>
