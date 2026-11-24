<script setup lang="ts">
/**
 * The vaults section of the plugin's settings: every vault of this device, whether it syncs, and the
 * two ways to connect one that does not.
 */
import { onMounted, onUnmounted } from 'vue'
import { Button, getCurrentDate } from '@numen/ui'
import type { VaultsClient } from '../api/vaults'
import { getErrorMessage, useVaults } from '../model/vaults'
import { WORDS as words } from '../words'

const REFRESH_MS = 15_000

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  client?: VaultsClient
  now?: () => Date
}>()

/* --------------------------------- State ---------------------------------- */
const model = useVaults({
  client: props.client,
  now: props.now ?? getCurrentDate,
})
const { status, error, isReachable, warning, rows, openFor, remote, pendingConfirm } = model.state
const {
  load,
  openRemote,
  closeRemote,
  connectNew,
  connectExisting,
  resolveDeletions,
  confirmConnect,
  cancelConnect,
} = model.actions

/* --------------------------------- Hooks ---------------------------------- */
let timer: ReturnType<typeof setInterval> | undefined

onMounted(() => {
  load().catch(report)
  timer = setInterval(() => {
    load().catch(report)
  }, REFRESH_MS)
})

onUnmounted(() => {
  clearInterval(timer)
})

/* -------------------------------- Handlers -------------------------------- */
function onConnectNew(vaultId: string): void {
  connectNew(vaultId).catch(report)
}

function onOpenRemote(vaultId: string): void {
  openRemote(vaultId).catch(report)
}

function onChooseRemote(vaultId: string, remoteId: string): void {
  connectExisting(vaultId, remoteId).catch(report)
}

function onConfirm(): void {
  confirmConnect().catch(report)
}

function onResolve(vaultId: string, choice: 'remove' | 'keep'): void {
  resolveDeletions(vaultId, choice).catch(report)
}

/* --------------------------------- Helpers -------------------------------- */
function report(failure: unknown): void {
  error.value = getErrorMessage(failure)
}
</script>

<template>
  <section class="vaults" :aria-label="words.vaultsTitle">
    <h2 class="vaults__title">{{ words.vaultsTitle }}</h2>
    <p v-if="status === 'loading'" class="vaults__note" role="status">{{ words.loading }}</p>
    <p v-if="warning !== ''" class="vaults__note vaults__note--alarm" role="status">
      {{ warning }}
    </p>
    <p v-if="error !== ''" class="vaults__note vaults__note--alarm" role="alert">{{ error }}</p>
    <ul class="vaults__list">
      <template v-for="row in rows" :key="row.id">
        <li class="vaults__row">
          <span class="vaults__name">{{ row.name }}</span>
          <span class="vaults__state">{{ row.stateLabel }}</span>
          <span class="vaults__path">{{ row.path }}</span>
          <span v-if="row.syncedLabel !== ''" class="vaults__detail">{{ row.syncedLabel }}</span>
          <span v-if="row.conflictLabel !== ''" class="vaults__detail vaults__detail--alarm">
            {{ row.conflictLabel }}
          </span>
          <Button
            v-if="row.canConnect"
            variant="outline"
            :disabled="!isReachable"
            @click="onConnectNew(row.id)"
          >
            {{ words.connectNew }}
          </Button>
          <Button
            v-if="row.canConnect"
            variant="outline"
            :disabled="!isReachable"
            @click="onOpenRemote(row.id)"
          >
            {{ words.connectExisting }}
          </Button>
        </li>
        <li v-if="row.pausedLabel !== ''" class="vaults__row vaults__row--paused">
          <span class="vaults__detail vaults__detail--alarm" role="status">
            {{ row.pausedLabel }}
          </span>
          <Button variant="outline" @click="onResolve(row.id, 'remove')">
            {{ words.removeFiles }}
          </Button>
          <Button variant="outline" @click="onResolve(row.id, 'keep')">
            {{ words.keepFiles }}
          </Button>
        </li>
        <li v-if="openFor === row.id" class="vaults__row vaults__row--remote">
          <span class="vaults__name">{{ words.chooseRemote }}</span>
          <span v-if="remote.length === 0" class="vaults__detail">{{ words.noRemoteVaults }}</span>
          <Button
            v-for="one in remote"
            :key="one.remoteId"
            variant="outline"
            :disabled="one.isConnected"
            @click="onChooseRemote(row.id, one.remoteId)"
          >
            {{ one.name }}
          </Button>
          <Button variant="ghost" @click="closeRemote">{{ words.closeRemote }}</Button>
        </li>
        <li
          v-if="pendingConfirm !== null && pendingConfirm.vaultId === row.id"
          class="vaults__row vaults__row--confirm"
        >
          <span class="vaults__detail vaults__detail--alarm" role="status">
            {{ pendingConfirm.text }}
          </span>
          <Button variant="outline" @click="onConfirm">{{ words.confirmConnect }}</Button>
          <Button variant="ghost" @click="cancelConnect">{{ words.cancelConnect }}</Button>
        </li>
      </template>
    </ul>
  </section>
</template>
