<script setup lang="ts">
/**
 * What is typed, and which step of a flow it is typed on.
 *
 * The field names the row that is lit for a screen reader; where that row is
 * and what the words turn up are the caller's.
 *
 * `data-palette` names each part: `crumb`, `field` and `busy`, the spinner at the end of
 * the row while more results are on their way.
 */
import { useTemplateRef } from 'vue'
import { Spinner } from '@numen/ui'
import { listId, optionId } from '../../lib/place'
import type { PaletteFieldProps } from './PaletteField.types'

defineProps<PaletteFieldProps>()

const searchQuery = defineModel<string>({ required: true })

defineExpose({ focus, select })

const field = useTemplateRef<HTMLInputElement>('field')

function focus(): void {
  field.value?.focus()
}

function select(): void {
  field.value?.select()
}
</script>

<template>
  <div class="palette__ask flex items-center gap-2">
    <!-- Which step the field is on. It says what the words typed here
         will mean. -->
    <span
      v-if="crumb"
      :id="`${uid}-crumb`"
      class="palette__crumb rounded-pill bg-bubble text-small px-2 py-0.5"
      data-palette="crumb"
    >
      {{ crumb }}
    </span>
    <input
      ref="field"
      v-model="searchQuery"
      class="palette__field w-full bg-transparent"
      data-palette="field"
      type="text"
      role="combobox"
      autocomplete="off"
      spellcheck="false"
      :placeholder="placeholder"
      :aria-label="name"
      :aria-describedby="crumb ? `${uid}-crumb` : undefined"
      :aria-expanded="isExpanded"
      :aria-busy="isBusy || undefined"
      :aria-controls="listId(uid)"
      :aria-activedescendant="here >= 0 ? optionId(uid, here) : undefined"
    />
    <!-- Over the end of the field, so the text and the row do not move. -->
    <Spinner v-if="isBusy" class="palette__busy" data-palette="busy" />
  </div>
</template>

<style scoped>
/* The padding at the start of the row; the step the field belongs to is laid out inside it. */
.palette__ask {
  position: relative;
  padding-inline-start: var(--numen-field-text-inset);
}

.palette__crumb {
  flex: none;
  max-inline-size: 45%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.palette__field {
  block-size: var(--numen-field-min);
  padding-inline-end: var(--numen-field-text-inset);
  border: 0;
  color: inherit;
  font: inherit;
  outline: none;
}

.palette__busy {
  position: absolute;
  inset-block-start: 50%;
  inset-inline-end: var(--numen-field-text-inset);
  translate: 0 -50%;
  pointer-events: none;
}

.palette__field::placeholder {
  color: var(--numen-edge-label);
}
</style>
