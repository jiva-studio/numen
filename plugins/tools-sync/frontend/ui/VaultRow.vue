<script setup lang="ts">
/**
 * A row of the card "Synced vaults": a square icon tinted with the state, the name, one quiet line that
 * starts with the state word, and what the person does with the vault on the right.
 */
import { computed } from 'vue'
import { Check, CircleAlert, CloudOff, RefreshCw } from '@lucide/vue'
import type { VaultRowKind } from '../model/vaults'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  name: string
  kind: VaultRowKind
  status: string
  details: readonly string[]
  progress?: number | null
}>()

/* --------------------------------- State ---------------------------------- */
const ICONS = { ok: Check, syncing: RefreshCw, off: CloudOff, conflict: CircleAlert } as const
const icon = computed(() => ICONS[props.kind])
const rest = computed(() => props.details.filter((one) => one !== ''))
const percent = computed(() => Math.round((props.progress ?? 0) * 100))
</script>

<template>
  <li class="vault-row" :class="`vault-row--${kind}`">
    <span class="vault-row__icon" aria-hidden="true">
      <component :is="icon" />
    </span>
    <div class="vault-row__label">
      <span class="vault-row__name">{{ name }}</span>
      <span class="vault-row__line" role="status">
        <span class="vault-row__state">{{ status }}</span>
        <template v-for="one in rest" :key="one"> · {{ one }}</template>
      </span>
      <span
        v-if="progress !== undefined && progress !== null"
        class="vault-row__progress"
        role="progressbar"
        :aria-label="words.syncProgressName"
        aria-valuemin="0"
        aria-valuemax="100"
        :aria-valuenow="percent"
      >
        <span class="vault-row__progress-fill" :style="{ inlineSize: `${percent}%` }" />
      </span>
    </div>
    <div v-if="$slots['actions']" class="vault-row__actions"><slot name="actions" /></div>
  </li>
</template>
