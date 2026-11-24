<script setup lang="ts">
/** The dialog of the command "Get a vault from the cloud...": it reads the vaults and draws the choice of one. */
import { onMounted } from 'vue'
import { getCurrentDate } from '@numen/ui'
import type { VaultsClient } from '../api/vaults'
import { getErrorMessage, useVaults } from '../model/vaults'
import CloudVaultDialog from './CloudVaultDialog.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ client?: VaultsClient }>()

const emit = defineEmits<{ (event: 'close'): void }>()

/* --------------------------------- State ---------------------------------- */
const { state, actions } = useVaults({ client: props.client, now: getCurrentDate })
const { cloudVaults, freeVaults, pendingConfirm, error } = state

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  actions.load().catch((failure: unknown) => {
    error.value = getErrorMessage(failure)
  })
})

/* -------------------------------- Handlers -------------------------------- */
function onChoose(vaultId: string, remoteId: string): void {
  actions.connectExisting(vaultId, remoteId).then(onDone).catch(onFail)
}

function onConfirm(): void {
  actions.confirmConnect().then(onDone).catch(onFail)
}

function onDone(): void {
  if (error.value === '' && pendingConfirm.value === null) emit('close')
}

function onFail(failure: unknown): void {
  error.value = getErrorMessage(failure)
}
</script>

<template>
  <CloudVaultDialog
    :cloud-vaults="cloudVaults"
    :free-vaults="freeVaults"
    :question="pendingConfirm?.text ?? ''"
    :problem="error"
    @choose="onChoose"
    @confirm="onConfirm"
    @back="actions.cancelConnect"
    @cancel="emit('close')"
  />
</template>
