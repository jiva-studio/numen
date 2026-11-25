<script setup lang="ts">
/**
 * A row of the card "Synced vaults": a square icon tinted with the state (ringed with the progress while it syncs), the name, the state word with its
 * qualifier, the facts on a quiet line of their own, and what the person does with the vault on the right.
 */
import { computed } from 'vue'
import { Check, CircleAlert, CloudOff, RefreshCw } from '@lucide/vue'
import type { VaultRowProps } from './VaultRow.types'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<VaultRowProps>()

/* --------------------------------- State ---------------------------------- */
const ICONS = { ok: Check, syncing: RefreshCw, off: CloudOff, conflict: CircleAlert } as const
const icon = computed(() => ICONS[props.kind])
const shownFacts = computed(() => props.facts.filter((one) => one !== ''))
const fraction = computed(() => Math.min(Math.max(props.progress ?? 0, 0), 1))
const percent = computed(() => Math.round(fraction.value * 100))
const isIndeterminate = computed(() => props.progress === undefined || props.progress === null)
const RING_RADIUS = 20
const RING_LENGTH = 2 * Math.PI * RING_RADIUS
const arc = computed(() => (isIndeterminate.value ? 0.25 : fraction.value) * RING_LENGTH)
</script>

<template>
  <li class="vault-row" :class="`vault-row--${kind}`">
    <span
      v-if="isTransferring"
      class="vault-row__icon vault-row__icon--ring"
      :class="{ 'vault-row__icon--indeterminate': isIndeterminate }"
      role="progressbar"
      :aria-label="words.syncProgressName"
      :aria-busy="isIndeterminate ? 'true' : undefined"
      :aria-valuemin="isIndeterminate ? undefined : 0"
      :aria-valuemax="isIndeterminate ? undefined : 100"
      :aria-valuenow="isIndeterminate ? undefined : percent"
    >
      <svg class="vault-row__ring" viewBox="0 0 44 44" aria-hidden="true">
        <circle class="vault-row__ring-track" cx="22" cy="22" :r="RING_RADIUS" />
        <circle
          class="vault-row__ring-arc"
          cx="22"
          cy="22"
          :r="RING_RADIUS"
          :stroke-dasharray="`${arc} ${RING_LENGTH}`"
        />
      </svg>
      <component :is="icon" class="vault-row__spin" aria-hidden="true" />
    </span>
    <span v-else class="vault-row__icon" aria-hidden="true">
      <component :is="icon" />
    </span>
    <div class="vault-row__label">
      <span class="vault-row__name">{{ name }}</span>
      <span class="vault-row__line vault-row__line--state" role="status">
        <span class="vault-row__piece vault-row__state">
          {{ qualifier === '' ? status : `${status} ·` }}
        </span>
        <span v-if="qualifier !== ''" class="vault-row__piece">{{ qualifier }}</span>
      </span>
      <span v-if="shownFacts.length > 0" class="vault-row__line vault-row__line--facts">
        <span v-for="(one, index) in shownFacts" :key="one" class="vault-row__piece">
          {{ index < shownFacts.length - 1 ? `${one} ·` : one }}
        </span>
      </span>
    </div>
    <div v-if="$slots['actions']" class="vault-row__actions"><slot name="actions" /></div>
  </li>
</template>
