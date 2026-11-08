<script setup lang="ts">
/**
 * What is typed, and which step of a flow it is typed on.
 *
 * The field names the row that is lit for a screen reader; where that row is
 * and what the words turn up are the caller's.
 *
 * `data-palette` names each part: `crumb` and `field`.
 */
import { useTemplateRef } from 'vue'
import { listId, optionId } from '../../lib/place'

defineProps<{
  uid: string
  here: number
  isExpanded: boolean
  placeholder: string
  name: string
  crumb: string
}>()

const searchQuery = defineModel<string>({ required: true })

const field = useTemplateRef<HTMLInputElement>('field')

const focus = (): void => field.value?.focus()
const select = (): void => field.value?.select()

defineExpose({ focus, select })
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
      :aria-controls="listId(uid)"
      :aria-activedescendant="here >= 0 ? optionId(uid, here) : undefined"
    />
  </div>
</template>

<style scoped>
/* The padding at the start of the row; the step the field belongs to is laid out inside it. */
.palette__ask {
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

.palette__field::placeholder {
  color: var(--numen-edge-label);
}
</style>
