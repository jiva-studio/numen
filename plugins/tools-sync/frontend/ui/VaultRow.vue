<script setup lang="ts">
/**
 * A row of the card "Synced vaults": a square icon tinted with the state (ringed with the progress while it syncs), the name, the state word with its
 * qualifier, the facts on a quiet line of their own, and what the person does with the vault on the right.
 */
import { computed } from 'vue'
import { Check, CircleAlert, CloudOff, RefreshCw } from '@lucide/vue'
import type { VaultRowProps } from './VaultRow.types'
import { WORDS as words } from '../words'
import SyncRing from './SyncRing.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<VaultRowProps>()

/* --------------------------------- State ---------------------------------- */
const ICONS = { ok: Check, syncing: RefreshCw, off: CloudOff, conflict: CircleAlert } as const
const icon = computed(() => ICONS[props.kind])
const shownFacts = computed(() => props.facts.filter((one) => one !== ''))
const ring = computed(() => props.progress ?? null)
const isIndeterminate = computed(() => ring.value === null)
const percent = computed(() => Math.round(Math.min(Math.max(ring.value ?? 0, 0), 1) * 100))
</script>

<template>
  <li class="vault-row" :class="`vault-row--${kind}`">
    <span
      v-if="isTransferring"
      class="vault-row__icon vault-row__icon--ring"
      role="progressbar"
      :aria-label="words.syncProgressName"
      :aria-busy="isIndeterminate ? 'true' : undefined"
      :aria-valuemin="isIndeterminate ? undefined : 0"
      :aria-valuemax="isIndeterminate ? undefined : 100"
      :aria-valuenow="isIndeterminate ? undefined : percent"
    >
      <SyncRing :progress="ring">
        <component :is="icon" class="sync-ring__glyph" aria-hidden="true" />
      </SyncRing>
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
