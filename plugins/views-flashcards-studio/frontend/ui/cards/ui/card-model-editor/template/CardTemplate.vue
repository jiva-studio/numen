<script setup lang="ts">
import { computed, nextTick, shallowRef, type ComponentPublicInstance } from 'vue'
import { CardHeader } from '../../card-header'
import { RemoveButton } from '../../remove-button'
import TemplatePane from './TemplatePane.vue'
import TemplateStrip from './TemplateStrip.vue'
import type { Half, StepDirection } from '../../../../../lib/cards'
import {
  getPanes,
  CARD_MODEL_WORDS,
  type TemplateRow,
  type Pane,
  type CardModelWords,
} from '../../../../../lib/cards'
import { insert } from '../../../../../lib/cards'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    template: TemplateRow
    wrong?: readonly string[]
    words?: CardModelWords
  }>(),
  { wrong: () => [], words: () => CARD_MODEL_WORDS },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'rename', name: string): void
  (event: 'remove'): void
  (event: 'drag-start', press: DragEvent): void
  (event: 'release'): void
  (event: 'step', direction: StepDirection, press: KeyboardEvent): void
  (event: 'write', half: Half, text: string): void
}>()

/* ---------------------------------- State --------------------------------- */
const aim = shallowRef<Half | null>(null)

const focusedHalf = computed<Half>(() => aim.value ?? 'front')

const front = shallowRef<InstanceType<typeof TemplatePane> | null>(null)
const back = shallowRef<InstanceType<typeof TemplatePane> | null>(null)

const divided = computed<readonly Pane[]>(() => getPanes(props.template, props.words))

/* -------------------------------- Handlers -------------------------------- */
const onSetBox = (pane: Pane, element: Element | ComponentPublicInstance | null): void => {
  if (pane.mode !== 'written') return
  const box = element as InstanceType<typeof TemplatePane> | null
  if (pane.half === 'front') front.value = box
  else back.value = box
}

const onPut = async (field: string): Promise<void> => {
  const half = focusedHalf.value
  const target = getBox(half)
  if (!target) return

  const caret = (aim.value === null ? null : target.selectionStart) ?? target.value.length
  const insertResult = insert(target.value, caret, field)
  emit('write', half, insertResult.text)
  await nextTick()
  const again = getBox(half)
  again?.focus()
  again?.setSelectionRange(insertResult.caret, insertResult.caret)
}

/* --------------------------------- Helpers -------------------------------- */
const getBox = (half: Half): HTMLTextAreaElement | null =>
  (half === 'front' ? front.value : back.value)?.box ?? null

const isFocused = (pane: Pane): boolean =>
  pane.mode === 'written' && focusedHalf.value === pane.half
</script>

<template>
  <article class="template rounded-node bg-raised flex flex-col" :data-template="template.id">
    <CardHeader
      :drag="`${words.drag}: ${template.name}`"
      @dragstart="emit('drag-start', $event)"
      @dragend="emit('release')"
      @step="(direction, press) => emit('step', direction, press)"
    >
      <TemplateStrip
        :template="template"
        :wrong="wrong"
        :words="words"
        @rename="(name) => emit('rename', name)"
        @put="onPut"
      />

      <template #actions>
        <RemoveButton :label="`${words.remove}: ${template.name}`" @press="emit('remove')" />
      </template>
    </CardHeader>

    <div class="template__body">
      <TemplatePane
        v-for="pane in divided"
        :key="`${pane.half}-${pane.mode}`"
        :ref="(el) => onSetBox(pane, el)"
        :pane="pane"
        :is-focused="isFocused(pane)"
        :words="words"
        @aim="aim = pane.half"
        @write="(text: string) => emit('write', pane.half, text)"
      />
    </div>
  </article>
</template>

<style scoped>
.template {
  --box-air: 0.5rem;
  --box-pad-inline: var(--numen-box-air);
  --pane-lines: 6;
  --pane-min: calc(var(--pane-lines) * var(--numen-line-height) * 1em + 2 * var(--box-air));

  position: relative;
  container-type: inline-size;
  inline-size: 100%;
  border: var(--numen-stroke) solid var(--numen-rule);
  overflow: hidden;
}

.template__body {
  display: grid;
  grid-template-columns: 1fr;
  gap: var(--numen-stroke);
  background: var(--numen-rule);
}

@container (min-width: 36rem) {
  .template__body {
    grid-template-columns: 1fr 1fr;
  }
}

.template :deep(.card-header) {
  position: static;
}
</style>
