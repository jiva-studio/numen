<script setup lang="ts">
/**
 * The panel a card is asked about in: the talk, and the field it is carried on
 * with.
 *
 * Why the agent cannot be reached is the window's to know, and it stands where
 * the answers do.
 */
import { Agent, Button, KeyCap } from '@numen/ui'

import { WORDS as words } from './agent/words'
import type { Held } from './asking'

const props = defineProps<{ held: Held; heading: string }>()
</script>

<template>
  <section class="asking" aria-label="Ask about this card">
    <header class="asking__where">
      <span class="asking__card">{{ props.heading }}</span>
      <Button variant="ghost" size="small" @click="props.held.shuts()">
        <KeyCap :keys="{ marks: [], letter: 'esc' }" />
        {{ words.shut }}
      </Button>
    </header>

    <!-- The field sends by a control of its own: enter in a panel inside a
         sitting would otherwise be read as an answer to the card. -->
    <Agent
      class="asking__talk"
      :model-value="props.held.written.value"
      :turns="props.held.turns.value"
      :working="props.held.working.value"
      :placeholder="words.ask"
      :sends="words.send"
      :stops="words.stop"
      @update:model-value="(text: string) => props.held.writing(text)"
      @submit="(text: string) => props.held.send(text)"
      @stop="props.held.stop()"
    >
      <template #silence>{{ props.held.unreachable() || words.nothingSaid }}</template>
      <template #failure="{ turn }">
        {{ turn.voice === 'asked' ? words.unsent : words.stopped }}
      </template>
    </Agent>
  </section>
</template>

<style scoped>
.asking {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-inline-size: 0;
  min-block-size: 0;
  gap: var(--numen-inset);
}

/* Which card is being asked about, said once and quietly: a person reading an
   answer is reading the answer. */
.asking__where {
  display: flex;
  align-items: center;
  gap: var(--numen-inset);
  color: var(--numen-hushed);
  font-size: var(--numen-font-size);
}

.asking__card {
  overflow: hidden;
  margin-inline-end: auto;
  color: var(--numen-node-fg);
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.asking__talk {
  flex: 1;
  min-block-size: 0;
}
</style>
