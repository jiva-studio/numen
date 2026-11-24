<script setup lang="ts">
/**
 * The dialog "Get a vault from the cloud": the vault of the account to get and the vault of this device that
 * receives it. When notes of the two differ, it asks once more before it connects them.
 */
import { computed, ref } from 'vue'
import { Button, ModalDialog } from '@numen/ui'
import type { RemoteVault, Vault } from '../api/vaults'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  cloudVaults: readonly RemoteVault[]
  freeVaults: readonly Vault[]
  question: string
}>()

const emit = defineEmits<{
  (event: 'choose', vaultId: string, remoteId: string): void
  (event: 'confirm'): void
  (event: 'back'): void
  (event: 'cancel'): void
}>()

/* --------------------------------- State ---------------------------------- */
const remoteId = ref(props.cloudVaults[0]?.remoteId ?? '')
const vaultId = ref(props.freeVaults[0]?.id ?? '')
const isAsking = computed(() => props.question !== '')
const canChoose = computed(() => remoteId.value !== '' && vaultId.value !== '')

/* -------------------------------- Handlers -------------------------------- */
function onSubmit(): void {
  if (isAsking.value) emit('confirm')
  else if (canChoose.value) emit('choose', vaultId.value, remoteId.value)
}

function onSecondary(): void {
  emit(isAsking.value ? 'back' : 'cancel')
}
</script>

<template>
  <ModalDialog
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
    <template #actions>
      <Button variant="ghost" @click="onSecondary">{{ words.cancel }}</Button>
      <Button type="submit" :disabled="!isAsking && !canChoose">{{ words.getVault }}</Button>
    </template>
  </ModalDialog>
</template>
