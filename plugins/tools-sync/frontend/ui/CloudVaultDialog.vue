<script setup lang="ts">
/**
 * The dialog "Get a vault from the cloud": the vault of the identity to get and the vault of this device that
 * receives it. When notes of the two differ, it asks once more before it connects them.
 */
import { computed, ref, useId } from 'vue'
import { Button, Dialog, Select } from '@numen/ui'
import type { RemoteVault, Vault } from '../api/vaults'
import { getChoice } from '../lib/choice'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  cloudVaults: readonly RemoteVault[]
  freeVaults: readonly Vault[]
  question: string
  problem?: string
}>()

const emit = defineEmits<{
  (event: 'choose', vaultId: string, remoteId: string): void
  (event: 'confirm'): void
  (event: 'back'): void
  (event: 'cancel'): void
}>()

/* --------------------------------- State ---------------------------------- */
const pickedRemote = ref('')
const pickedVault = ref('')
const remoteId = computed({
  get: () =>
    getChoice(
      pickedRemote.value,
      props.cloudVaults.map((one) => one.remoteId),
    ),
  set: (next: string) => (pickedRemote.value = next),
})
const vaultId = computed({
  get: () =>
    getChoice(
      pickedVault.value,
      props.freeVaults.map((one) => one.id),
    ),
  set: (next: string) => (pickedVault.value = next),
})
const cloudLabelId = useId()
const deviceLabelId = useId()
const cloudChoices = computed(() =>
  props.cloudVaults.map((one) => ({ id: one.remoteId, text: one.name })),
)
const deviceChoices = computed(() =>
  props.freeVaults.map((one) => ({ id: one.id, text: one.name })),
)
const isAsking = computed(() => props.question !== '')
const canChoose = computed(() => remoteId.value !== '' && vaultId.value !== '')

/* -------------------------------- Handlers -------------------------------- */
function onSubmit(): void {
  if (isAsking.value) emit('confirm')
  else if (canChoose.value) emit('choose', vaultId.value, remoteId.value)
}

function onSecondary(): void {
  if (isAsking.value) emit('back')
  else emit('cancel')
}
</script>

<template>
  <Dialog
    is-form
    :label="words.getFromCloud"
    :hint="isAsking ? '' : words.getFromCloudText"
    :step="isAsking ? 'ask' : 'choose'"
    @submit="onSubmit"
    @cancel="emit('cancel')"
  >
    <p v-if="isAsking" class="sync-dialog__text" role="status">{{ question }}</p>
    <template v-else>
      <div class="sync-field">
        <span :id="cloudLabelId" class="sync-field__label">{{ words.cloudVault }}</span>
        <Select
          v-model="remoteId"
          :choices="cloudChoices"
          :name="words.cloudVault"
          :aria-labelledby="cloudLabelId"
          is-searchable
        />
      </div>
      <div class="sync-field">
        <span :id="deviceLabelId" class="sync-field__label">{{ words.deviceVault }}</span>
        <Select
          v-model="vaultId"
          :choices="deviceChoices"
          :name="words.deviceVault"
          :aria-labelledby="deviceLabelId"
          :is-disabled="freeVaults.length === 0"
          is-searchable
        />
      </div>
      <p v-if="freeVaults.length === 0" class="sync-dialog__text" role="status">
        {{ words.noDeviceVault }}
      </p>
    </template>
    <p v-if="problem" class="sync-dialog__text sync-dialog__text--alarm" role="alert">
      {{ problem }}
    </p>
    <template #actions>
      <Button type="button" variant="ghost" @click="onSecondary">{{ words.cancel }}</Button>
      <Button type="submit" :disabled="!isAsking && !canChoose">{{ words.getVault }}</Button>
    </template>
  </Dialog>
</template>
