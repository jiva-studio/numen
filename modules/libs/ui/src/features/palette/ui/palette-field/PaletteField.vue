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
  /**
   * What the field and the list beside it are addressed by. The field names the
   * row that is lit, so the two are addressed off one prefix.
   */
  uid: string
  /** The number of the row the keyboard stands on, and -1 for none. */
  here: number
  /** Whether there is a list under the field at all. */
  expanded: boolean
  /** The words standing in for what has not been typed. */
  placeholder: string
  /** What the field is announced as. */
  name: string
  /**
   * Which step of a flow the field is on, in a few words, drawn beside it.
   * Nothing is drawn without it.
   */
  crumb: string
}>()

const typed = defineModel<string>({ required: true })

const field = useTemplateRef<HTMLInputElement>('field')

/** The keyboard into the field, and what stands there selected. */
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
      class="palette__crumb rounded-pill bg-bubble px-2 py-0.5 text-small"
      data-palette="crumb"
      >{{ crumb }}</span
    >
    <input
      ref="field"
      v-model="typed"
      class="palette__field w-full bg-transparent"
      data-palette="field"
      type="text"
      role="combobox"
      autocomplete="off"
      spellcheck="false"
      :placeholder="placeholder"
      :aria-label="name"
      :aria-describedby="crumb ? `${uid}-crumb` : undefined"
      :aria-expanded="expanded"
      :aria-controls="listId(uid)"
      :aria-activedescendant="here >= 0 ? optionId(uid, here) : undefined"
    />
  </div>
</template>

<style scoped>
/* The clearance at the head of the row, which the step the field is on stands
   inside. */
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
