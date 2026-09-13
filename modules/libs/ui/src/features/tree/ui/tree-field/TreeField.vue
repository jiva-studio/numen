<script setup lang="ts">
/**
 * A name being typed, over the row it belongs to.
 *
 * It keeps the keyboard and the pointer to itself, so nothing done in it
 * reaches the row underneath. Which row is being renamed, and where the
 * keyboard goes once the typing is over, are the tree's.
 */
import { useTemplateRef } from 'vue'

defineProps<{
  /** The name as it stands, which is what is typed over. */
  value: string
  /** What the field is announced as. */
  name: string
}>()

const emit = defineEmits<{
  /** A name typed and committed. */
  (event: 'rename', name: string): void
  /** The name left as it was, and the keyboard asked back to the row. */
  (event: 'abandon'): void
  /** The field left, the keyboard already somewhere else. */
  (event: 'blur'): void
}>()

const field = useTemplateRef<HTMLInputElement>('field')

/** The keyboard onto the field, with the name in it ready to be replaced. */
const focus = (): void => {
  field.value?.focus()
  field.value?.select()
}

defineExpose({ focus })

function onKey(event: KeyboardEvent): void {
  if (event.key === 'Enter') {
    event.preventDefault()
    emit('rename', (event.currentTarget as HTMLInputElement).value)
    return
  }

  if (event.key === 'Escape') {
    event.preventDefault()
    emit('abandon')
  }
}
</script>

<template>
  <input
    ref="field"
    class="tree__field rounded-node min-w-0 grow"
    type="text"
    :value="value"
    :aria-label="name"
    @pointerdown.stop
    @click.stop
    @dblclick.stop
    @keydown.stop="onKey"
    @blur="emit('blur')"
  />
</template>

<style scoped>
.tree__field {
  border: var(--numen-stroke) solid var(--numen-field-border);
  background: var(--numen-field-bg);
  color: var(--numen-ink);
  font: inherit;
}

/* Where the keyboard stands. */
.tree__field:focus-visible {
  outline: none;
}
</style>
