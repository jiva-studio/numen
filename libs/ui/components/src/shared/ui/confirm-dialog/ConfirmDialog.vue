<script setup lang="ts">
/**
 * A question over the tab that is answered with one of two buttons, with a title and a text, or with what the
 * default slot draws in place of the text. Focus starts on the cancel button; Escape, a press on the backdrop and
 * the cancel button all emit `cancel`, and Tab stays between the two buttons.
 */
import { onMounted, useId, useTemplateRef } from 'vue'
import { Button } from '../button'
import { ModalDialog } from '../modal-dialog'

/* --------------------------------- Props ---------------------------------- */
withDefaults(
  defineProps<{
    text?: string
    label: string
    confirmText: string
    cancelText: string
    tone?: 'danger' | 'default'
  }>(),
  { text: '', tone: 'default' },
)

const emit = defineEmits<{
  (e: 'confirm'): void
  (e: 'cancel'): void
}>()

/* ---------------------------------- State --------------------------------- */
const textId = useId()
const cancel = useTemplateRef<InstanceType<typeof Button>>('cancel')

/* ---------------------------------- Hooks --------------------------------- */
onMounted(() => {
  const element = cancel.value?.$el
  if (element instanceof HTMLElement) element.focus()
})
</script>

<template>
  <ModalDialog
    role="alertdialog"
    :label="label"
    :tone="tone"
    :described-by="textId"
    @cancel="emit('cancel')"
  >
    <div :id="textId" class="confirm-dialog__text">
      <slot>{{ text }}</slot>
    </div>
    <template #actions>
      <Button ref="cancel" variant="ghost" @click="emit('cancel')">{{ cancelText }}</Button>
      <Button :variant="tone === 'danger' ? 'danger' : 'solid'" @click="emit('confirm')">
        {{ confirmText }}
      </Button>
    </template>
  </ModalDialog>
</template>
