<script setup lang="ts">
/**
 * The secret one declaration of a plugin reads: a masked field that shows no value and takes none,
 * and an arrow that opens the secrets to choose from. A click on a secret allows the declaration
 * to read it, or stops it, as the check box on the page of secrets does. Where the declared group
 * has no secret, a button opens the page of secrets.
 */
import { computed, ref, useTemplateRef } from 'vue'
import { Check, ChevronDown } from '@lucide/vue'
import { Button, SearchableListPopup, measureRect, type ListItem } from '@numen/ui'
import type { Secret } from '@/entities/settings'
import {
  getMenuSecrets,
  hasGroupSecrets,
  isAllowed,
  type Declaration,
  type SecretSection,
} from '../../lib/secretSections'
import { WORDS as words } from '../../words'
import SettingRow from '../setting-row/SettingRow.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  at: string
  name: string
  detail: string
  declaration: Declaration
  sections: readonly SecretSection[]
}>()

const emit = defineEmits<{
  (e: 'toggle', secret: Secret, isAllowed: boolean): void
  (e: 'open-secrets'): void
}>()

/* --------------------------------- State ---------------------------------- */
const trigger = useTemplateRef<HTMLElement>('trigger')
const anchor = ref<{ x: number; y: number; width: number } | null>(null)
const query = ref('')

const secrets = computed(() => getMenuSecrets(props.sections, props.declaration))
const items = computed<readonly ListItem[]>(() =>
  secrets.value.map((one) => ({ id: one.id, text: one.name })),
)
const allowedIds = computed(
  () =>
    new Set(secrets.value.filter((one) => isAllowed(one, props.declaration)).map((one) => one.id)),
)
const summary = computed(() => {
  const names = secrets.value.filter((one) => allowedIds.value.has(one.id)).map((one) => one.name)
  return names.length === 0 ? words.secretFieldNone : names.join(', ')
})
const hasOwnSecrets = computed(() => hasGroupSecrets(props.sections, props.declaration))
const groupOf = (id: string): string => secrets.value.find((one) => one.id === id)?.group ?? ''

/* -------------------------------- Handlers -------------------------------- */
function onOpen(): void {
  const box = measureRect(trigger.value)
  anchor.value = { x: box?.left ?? 0, y: box?.bottom ?? 0, width: box?.width ?? 0 }
}

function onClose(): void {
  anchor.value = null
  query.value = ''
}

function onChoose(id: string): void {
  const secret = secrets.value.find((one) => one.id === id)
  if (secret) emit('toggle', secret, !allowedIds.value.has(id))
}
</script>

<template>
  <SettingRow v-slot="{ labelledBy }" :at="at" :name="name" :detail="detail">
    <span ref="trigger" class="settings__secret">
      <input
        type="password"
        readonly
        autocomplete="off"
        :placeholder="summary"
        :aria-labelledby="labelledBy"
        class="settings__secret-input"
        @click="onOpen"
      />
      <Button
        variant="ghost"
        size="icon-small"
        class="settings__secret-choose"
        aria-haspopup="listbox"
        :aria-expanded="anchor !== null"
        :aria-label="words.secretChoose"
        :title="words.secretChoose"
        @click="onOpen"
      >
        <ChevronDown :size="14" aria-hidden="true" />
      </Button>
      <Button
        v-if="!hasOwnSecrets && declaration.group !== ''"
        variant="outline"
        size="small"
        @click="emit('open-secrets')"
      >
        {{ words.secretOpenPage }}
      </Button>
    </span>

    <SearchableListPopup
      v-if="anchor"
      v-model:query="query"
      :items="items"
      :label="words.secretChoose"
      :placeholder="words.secretSearch"
      :empty-text="words.secretNothing"
      :at="{ x: anchor.x, y: anchor.y }"
      :asking="anchor.width"
      :from="trigger"
      @select="onChoose"
      @close="onClose"
    >
      <template #leading="{ item }">
        <Check v-if="allowedIds.has(item.id)" :size="14" aria-hidden="true" />
      </template>
      <template #trailing="{ item }">
        <span class="settings__secret-group">{{ groupOf(item.id) }}</span>
      </template>
    </SearchableListPopup>
  </SettingRow>
</template>

<style scoped>
.settings__secret {
  position: relative;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.375rem;
  inline-size: var(--settings-choice);
  min-inline-size: 0;
  max-inline-size: 100%;
}

.settings__secret-input {
  box-sizing: border-box;
  inline-size: var(--settings-choice);
  block-size: var(--numen-action-size);
  flex: 1 1 100%;
  min-inline-size: 0;
  max-inline-size: 24rem;
  padding: 0.25rem 2rem 0.25rem 0.625rem;
  background-color: var(--numen-field-bg);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  color: var(--numen-ink);
  cursor: pointer;
  font-family: inherit;
  font-size: var(--numen-text-2);
}

.settings__secret-input:focus {
  outline: var(--numen-stroke) solid var(--numen-accent);
}

.settings__secret-choose {
  position: absolute;
  inset-block-start: calc((var(--numen-action-size) - 1.5rem) / 2);
  inset-inline-end: 0.25rem;
  inline-size: 1.5rem;
  block-size: 1.5rem;
  color: var(--numen-hushed);
}

.settings__secret-group {
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}
</style>
