<script setup lang="ts">
/**
 * The box a name is typed in, over the name something already carries.
 *
 * It carries neither a line nor a ground of its own, so it reads as the name
 * itself and not as a thing to fill in. What is wrong with what is typed is
 * said by whatever draws this, under the row the box stands in.
 */
import type { NamingState } from '../naming'

const props = defineProps<{
  /** The naming this box types into. */
  naming: NamingState<unknown>
  /** What is being typed over, as the naming addresses it. */
  over: string
  /** What the box is announced by, and what stands in it while it is empty. */
  stem: string
  /** What says what is wrong with the name, where something does. */
  describedBy?: string | null
}>()

/** Whether what is typed cannot be used. */
const objects = (): boolean => props.naming.objection(props.over) !== null
</script>

<template>
  <!-- A box asked for one character takes the width whatever holds it gives it. -->
  <input
    class="name-box"
    type="text"
    size="1"
    :value="naming.text(over)"
    :placeholder="stem"
    :aria-label="stem"
    :aria-invalid="objects() || undefined"
    :aria-describedby="describedBy ?? undefined"
    @input="naming.typing(over, ($event.target as HTMLInputElement).value)"
    @change="naming.commit(over)"
    @keydown="naming.onKey($event, over)"
  />
</template>

<style scoped>
/* The air a row keeps at its ends, where the box stands in one. */
.name-box {
  padding: var(--card-row-pad-block, 0.125rem) var(--card-row-pad-inline, 0.375rem);
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
