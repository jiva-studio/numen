<script setup lang="ts">
/**
 * The question before a vault stops syncing or a received folder is left: the files stay on this device either way.
 * A synced vault also asks whether its copy in the cloud stays or is deleted.
 */
import { onMounted, useId, useTemplateRef } from 'vue'
import { Button, ConfirmDialog, Dialog } from '@numen/ui'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
withDefaults(defineProps<{ kind: 'stop' | 'leave'; name?: string }>(), { name: '' })

const emit = defineEmits<{
  (event: 'confirm', deleteCloudCopy: boolean): void
  (event: 'cancel'): void
}>()

/* --------------------------------- State ---------------------------------- */
const textId = useId()
const cancel = useTemplateRef<InstanceType<typeof Button>>('cancel')

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  const element = cancel.value?.$el
  if (element instanceof HTMLElement) element.focus()
})
</script>

<template>
  <Dialog
    v-if="kind === 'stop'"
    role="alertdialog"
    :label="words.stopSyncingQuestion(name)"
    :described-by="textId"
    @cancel="emit('cancel')"
  >
    <p :id="textId" class="sync-dialog__text">{{ words.stopSyncingText }}</p>
    <template #actions>
      <Button ref="cancel" variant="ghost" @click="emit('cancel')">{{ words.cancel }}</Button>
      <Button variant="solid" @click="emit('confirm', false)">{{ words.keepCloudCopy }}</Button>
      <Button variant="danger" @click="emit('confirm', true)">{{ words.deleteCloudCopy }}</Button>
    </template>
  </Dialog>
  <ConfirmDialog
    v-else
    :label="words.leaveTitle"
    :text="words.leaveText"
    :confirm-text="words.leaveShare"
    :cancel-text="words.cancel"
    @confirm="emit('confirm', false)"
    @cancel="emit('cancel')"
  />
</template>
