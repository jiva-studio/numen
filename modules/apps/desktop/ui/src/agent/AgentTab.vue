<script setup lang="ts">
/**
 * An agent tab: one thread of talk, and the composer it is asked in.
 *
 * Why the agent cannot be reached is the window's to know, and it stands where
 * the answers do.
 */
import { Agent } from '@numen/ui'
import type { Turn } from '@numen/ui'
import { WORDS as words } from '../words'
import type { Held } from './kind'

const props = defineProps<{ held: Held }>()
</script>

<template>
  <Agent
    :model-value="props.held.asked.value"
    :turns="props.held.turns.value"
    :working="props.held.working.value"
    :placeholder="words.ask"
    :sends="words.send"
    :stops="words.stop"
    @update:model-value="(text: string) => props.held.writing(text)"
    @submit="(text: string) => props.held.send(text)"
    @stop="props.held.stop()"
    @open="(turn: Turn) => props.held.opensTurn(turn)"
    @follow="
      (turn: Turn, href: string, press: MouseEvent) => props.held.followed(turn, href, press)
    "
  >
    <template #silence>{{ props.held.unreachable() || words.nothingSaid }}</template>
    <template #failure="{ turn }">
      {{ turn.voice === 'asked' ? words.unsent : words.stopped }}
    </template>
  </Agent>
</template>
