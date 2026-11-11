<script setup lang="ts">
import { computed } from 'vue'
import type { NameEntryState } from '../../../model/cards/nameEntry'
import { useObjection } from '../../../model/cards/useObjection'
import type { Objection } from '../../../../lib/cards'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  naming: NameEntryState<Objection>
  targetId: string
  stem: string
  describedBy?: string | null
}>()

/* --------------------------------- State ---------------------------------- */
const objection = useObjection(
  () => props.naming,
  () => props.targetId,
)
const isInvalid = computed<boolean>(() => objection.value !== null)

/* -------------------------------- Handlers -------------------------------- */
function onInput(event: Event): void {
  const target = event.target as HTMLInputElement
  props.naming.setDraft(props.targetId, target.value)
}

function onChange(): void {
  props.naming.commit(props.targetId)
}

function onKeydown(event: KeyboardEvent): void {
  props.naming.onKey(event, props.targetId)
}
</script>

<template>
  <input
    class="name-box"
    type="text"
    size="1"
    :value="naming.getText(targetId)"
    :placeholder="stem"
    :aria-label="stem"
    :aria-invalid="isInvalid || undefined"
    :aria-describedby="describedBy ?? undefined"
    @input="onInput"
    @change="onChange"
    @keydown="onKeydown"
  />
</template>

<style scoped>
/* Input padding when rendered inside a card row container. */
.name-box {
  padding: var(--card-row-pad-block) var(--card-row-pad-inline);
  border: none;
  background: none;
  color: inherit;
  font: inherit;
  cursor: auto;
}

.name-box:focus-visible {
  outline: none;
}
</style>
