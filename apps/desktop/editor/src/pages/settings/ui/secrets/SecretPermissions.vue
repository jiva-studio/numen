<script setup lang="ts">
/**
 * The plugin declarations that may read a secret: a button that reads the number of plugins and
 * opens a searchable list of the declarations, with a check mark on each one that is allowed.
 * A click on a row allows the declaration to read the secret, or stops it.
 */
import { computed, ref, useTemplateRef } from 'vue'
import { Check, ChevronDown } from '@lucide/vue'
import { SearchableListPopup, measureRect, type ListItem } from '@numen/ui'
import type { Secret } from '@/entities/settings'
import { isAllowed, type Declaration } from '../../lib/secretSections'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  /** Absent while the secret does not exist yet: the list is then disabled. */
  secret?: Secret | undefined
  declarations: readonly Declaration[]
}>()

const emit = defineEmits<{
  (e: 'toggle', declaration: Declaration, isAllowed: boolean): void
}>()

/* --------------------------------- State ---------------------------------- */
const trigger = useTemplateRef<HTMLElement>('trigger')
const anchor = ref<{ x: number; y: number; width: number } | null>(null)
const query = ref('')

const keyOf = (one: Declaration): string => `${one.pluginId}/${one.name}/${one.group}`
const items = computed<readonly ListItem[]>(() =>
  props.declarations.map((one) => ({ id: keyOf(one), text: one.pluginName })),
)
const allowedIds = computed(() => {
  const secret = props.secret
  const allowed = props.declarations.filter((one) => secret !== undefined && isAllowed(secret, one))
  return new Set(allowed.map(keyOf))
})
const count = computed(() => allowedIds.value.size)

/* -------------------------------- Handlers -------------------------------- */
function onClose(): void {
  anchor.value = null
  query.value = ''
}

function onOpen(): void {
  if (anchor.value) return onClose()
  const box = measureRect(trigger.value)
  anchor.value = { x: box?.left ?? 0, y: box?.bottom ?? 0, width: box?.width ?? 0 }
}

function onChoose(id: string): void {
  const declaration = props.declarations.find((one) => keyOf(one) === id)
  if (declaration) emit('toggle', declaration, !allowedIds.value.has(id))
}
</script>

<template>
  <div class="secret-permissions">
    <button
      ref="trigger"
      type="button"
      class="secret-permissions__button"
      aria-haspopup="listbox"
      :disabled="!secret || declarations.length === 0"
      :title="secret ? undefined : words.secretSaveFirst"
      :aria-expanded="anchor !== null"
      @click="onOpen"
    >
      {{ words.secretReaders(count) }}
      <ChevronDown :size="14" aria-hidden="true" />
    </button>
    <SearchableListPopup
      v-if="anchor"
      :model-value="null"
      v-model:query="query"
      :items="items"
      :label="words.secretAllowedTo"
      :placeholder="words.secretSearchPlugins"
      :empty-text="words.secretNoPlugins"
      :at="{ x: anchor.x, y: anchor.y }"
      :asking="anchor.width"
      :from="trigger"
      @select="onChoose"
      @close="onClose"
    >
      <template #leading="{ item }">
        <Check
          class="secret-permissions__check"
          :class="{ 'secret-permissions__check--off': !allowedIds.has(item.id) }"
          :size="14"
          aria-hidden="true"
        />
      </template>
    </SearchableListPopup>
  </div>
</template>

<style scoped>
.secret-permissions__button {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.25rem;
  box-sizing: border-box;
  inline-size: 100%;
  block-size: var(--numen-action-size);
  padding: 0.25rem 0.625rem;
  background-color: var(--numen-field-bg);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  color: var(--numen-ink);
  font-family: inherit;
  font-size: var(--numen-text-2);
  white-space: nowrap;
  cursor: pointer;
}

.secret-permissions__button:disabled {
  color: var(--numen-hushed);
  cursor: default;
}

.secret-permissions__check {
  flex: none;
}

.secret-permissions__check--off {
  visibility: hidden;
}
</style>
