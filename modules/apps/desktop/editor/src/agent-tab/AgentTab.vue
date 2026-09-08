<script setup lang="ts">
/**
 * An agent tab: one thread of talk, and the composer it is asked in.
 *
 * Why the agent cannot be reached is the window's to know, and it stands where
 * the answers do.
 */
import { Agent } from '@numen/ui'
import type { Turn } from '@numen/ui'
import { WORDS as words } from './words'
import type { AgentTabState } from './kind'

const props = defineProps<{ state: AgentTabState }>()
</script>

<template>
  <Agent
    :model-value="props.state.asked.value"
    :turns="props.state.turns.value"
    :working="props.state.working.value"
    :placeholder="words.ask"
    :sends="words.send"
    :stops="words.stop"
    @update:model-value="(text: string) => props.state.writing(text)"
    @submit="(text: string) => props.state.send(text)"
    @stop="props.state.stop()"
    @open="(turn: Turn) => props.state.opensTurn(turn)"
    @follow="
      (turn: Turn, href: string, press: MouseEvent) => props.state.followed(turn, href, press)
    "
  >
    <template #silence>{{ props.state.unreachable() || words.nothingSaid }}</template>
    <template #failure="{ turn }">
      {{ turn.voice === 'asked' ? words.unsent : words.stopped }}
    </template>
  </Agent>
</template>
