<script setup lang="ts">
/**
 * The dialog "Get a vault from the cloud": the vault of the identity to get and the vault of this device that
 * receives it. When notes of the two differ, it asks once more before it connects them.
 */
import { computed, ref } from 'vue'
import { Button, Dialog } from '@numen/ui'
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
      <label class="sync-field">
        <span class="sync-field__label">{{ words.cloudVault }}</span>
        <select v-model="remoteId" class="sync-field__input">
          <option v-for="one in cloudVaults" :key="one.remoteId" :value="one.remoteId">
            {{ one.name }}
          </option>
        </select>
      </label>
      <label class="sync-field">
        <span class="sync-field__label">{{ words.deviceVault }}</span>
        <select v-model="vaultId" class="sync-field__input" :disabled="freeVaults.length === 0">
          <option v-for="one in freeVaults" :key="one.id" :value="one.id">{{ one.name }}</option>
        </select>
      </label>
      <p v-if="freeVaults.length === 0" class="sync-dialog__text" role="status">
        {{ words.noDeviceVault }}
      </p>
    </template>
    <p v-if="problem" class="sync-dialog__text sync-dialog__text--alarm" role="alert">
      {{ problem }}
    </p>
    <template #actions>
      <Button variant="ghost" @click="onSecondary">{{ words.cancel }}</Button>
      <Button type="submit" :disabled="!isAsking && !canChoose">{{ words.getVault }}</Button>
    </template>
  </Dialog>
</template>
