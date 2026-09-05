<script setup lang="ts">
/**
 * The panel a card is asked about in: the talk, and the field it is carried on
 * with.
 *
 * Why the agent cannot be reached is the window's to know, and it stands where
 * the answers do.
 */
import { nextTick, useTemplateRef, watch } from 'vue'
import { Agent } from '@numen/ui'

import { WORDS as words } from './agent/words'
import type { Held } from './asking'

const props = defineProps<{ held: Held }>()

const talk = useTemplateRef<InstanceType<typeof Agent>>('talk')

// A panel opened is a panel opened to write in, so the field takes the keyboard
// as it arrives. Taking it scrolls nothing: the panel is arriving, and a
// browser bringing the field into view would drag what is moving.
watch(
  () => props.held.open.value,
  (up) => {
    if (up) void nextTick(() => talk.value?.focus({ preventScroll: true }))
  },
)
</script>

<template>
  <section class="asking" aria-label="Ask about this card">
    <!-- The field sends by a control of its own: enter in a panel inside a
         sitting would otherwise be read as an answer to the card. -->
    <Agent
      ref="talk"
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
/* The panel stands where the card stood and is the same thing to look at, so it
   takes the card's ground. */
.asking {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-inline-size: 0;
  min-block-size: 0;
  padding: var(--numen-inset-wide);
  border: 1px solid var(--numen-rule);
  border-radius: var(--numen-radius);
  background: var(--numen-raised);
}

.asking__talk {
  flex: 1;
  min-block-size: 0;
}
</style>
