<script setup lang="ts">
/**
 * Agent conversation tab with message thread and composer.
 */
import { Agent } from '@numen/ui'
import type { Turn } from '@numen/ui'
import { WORDS as words } from './words'
import type { AgentTabState } from './kind'

// --- Props & Emits ---
const props = defineProps<{ state: AgentTabState }>()

// --- State ---

// --- Handlers ---
function onUpdateQuestion(text: string) {
  props.state.setQuestion(text)
}

function onSubmit(text: string) {
  props.state.send(text)
}

function onStop() {
  props.state.stop()
}

function onOpenTurn(turn: Turn) {
  props.state.openTurnSource(turn)
}

function onFollowLink(turn: Turn, href: string, press: MouseEvent) {
  props.state.followLink(turn, href, press)
}

// --- Helpers ---
</script>

<template>
  <Agent
    :model-value="props.state.asked.value"
    :turns="props.state.turns.value"
    :working="props.state.working.value"
    :placeholder="words.ask"
    :sends="words.send"
    :stops="words.stop"
    @update:model-value="onUpdateQuestion"
    @submit="onSubmit"
    @stop="onStop"
    @open="onOpenTurn"
    @follow="onFollowLink"
  >
    <template #silence>{{ props.state.unreachable() || words.nothingSaid }}</template>
    <template #failure="{ turn }">
      {{ turn.voice === 'asked' ? words.unsent : words.stopped }}
    </template>
  </Agent>
</template>
