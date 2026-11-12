<script setup lang="ts">
/**
 * A name being typed, over the row it belongs to.
 *
 * It captures the keyboard and the pointer, so no input in it propagates to the
 * row underneath. The tree decides which row is being renamed and where
 * keyboard focus goes after the typing ends.
 */
import { useTemplateRef } from 'vue'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    value: string
    name: string
    isFolder?: boolean
  }>(),
  {
    isFolder: false,
  },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  /** A name typed and committed. */
  (event: 'rename', name: string): void
  /** The name is left unchanged, and keyboard focus returns to the row. */
  (event: 'abandon'): void
  /** The field left, the keyboard already somewhere else. */
  (event: 'blur'): void
}>()

defineExpose({ focus })

/* --------------------------------- State ---------------------------------- */
const field = useTemplateRef<HTMLInputElement>('field')

/* -------------------------------- Handlers -------------------------------- */
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

/* -------------------------------- Helpers --------------------------------- */
function focus(): void {
  const input = field.value
  if (!input) return
  input.focus()

  const value = input.value
  if (props.isFolder) {
    input.select()
    return
  }

  const lastDot = value.lastIndexOf('.')
  if (lastDot <= 0) {
    input.select()
    return
  }

  input.setSelectionRange(0, lastDot)
}
</script>

<template>
  <input
    ref="field"
    class="tree__field min-w-0 grow"
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
  /* The field is drawn two hairlines inside the row, and reaches out to the left to
     keep its text over the name it replaces. */
  --trim: 0.125rem;
  --reach: 0.1875rem;
  --pad: 0.125rem;
  --corner: 0.0625rem;

  box-sizing: border-box;
  inline-size: 100%;
  block-size: calc(var(--row) - var(--trim));
  margin-inline-start: calc(-1 * var(--reach));
  padding-inline: var(--pad);
  padding-block: 0;
  border: var(--numen-stroke) solid var(--numen-ring);
  border-radius: var(--corner);
  background: var(--numen-raised);
  color: var(--numen-ink);
  font: inherit;
  line-height: inherit;
}

/* Keyboard focus. */
.tree__field:focus-visible {
  outline: none;
}
</style>
