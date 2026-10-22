<script setup lang="ts">
/**
 * The screen a window holding nothing shows: the vaults this installation
 * holds, and the ways into the one it is showing.
 *
 * chosen here, where the window's own icons are. A letter alone opens the vault
 * standing at it, which is the letter drawn on that row.
 */
import { computed, onMounted, onUnmounted } from 'vue'
import { getVaultForKey, WelcomePage } from '@numen/ui'
import type { Tab } from '@numen/ui'
import {
  runInvocation,
  createInvocation,
  createRowKeys,
  ANSWER_WORDS,
  type CommandDeps,
  type Commands,
  type CommandTarget,
  type SearchState,
  type VaultRef,
} from '@/features/command-palette'
import type { VaultList } from '@/entities/vault'
import { findIcon } from '@/shared/icons'
import { VERSION } from '../lib/version'
import { COMMANDS, createVaultRows, createWelcomeActions } from '../lib/screen'
import { WORDS as words } from '@/shared/words'

// --- Props & Emits ---
const props = defineProps<{
  /** Every vault the installation holds, as the list last answered. */
  vaultList: VaultList
  /** Every tab the window holds, which is none while this screen is what shows. */
  tabs: readonly Tab[]
  commands: Commands
  search: SearchState
  doing: CommandDeps
  getTarget: () => CommandTarget
  /** A command asked for, which is what every way in but the commands evaluates to. */
  runCommand: (id: string, target: CommandTarget) => void
}>()

// --- State ---
const adding = computed(() => {
  const icon = findIcon('newVault')
  return {
    text: words.newVault,
    detail: words.newVaultDetail,
    ...(icon ? { icon } : {}),
    ...createRowKeys('newVault', navigator.userAgent),
  }
})

const ways = computed(() => {
  const target = props.getTarget()
  return createWelcomeActions(
    { vault: target.vault.id, isReady: target.isReady },
    words,
    navigator.userAgent,
  ).map((one) => {
    const icon = findIcon(one.id)
    return { ...one, ...(icon ? { icon } : {}) }
  })
})

const onList = computed(() => createVaultRows(props.vaultList, words))

/** Whether the welcome screen is what the person is looking at and typing into. */
const welcoming = computed(
  () => props.tabs.length === 0 && !props.search.open.value && !props.commands.open.value,
)

// --- Handlers ---
function onRun(id: string) {
  if (id !== COMMANDS) return props.runCommand(id, props.getTarget())
  props.search.setOpen(false)
  props.commands.setOpen(true)
}

function onOpen(id: string) {
  const one = props.vaultList.vaults.find((vault) => vault.id === id)
  if (!one) return
  const vault: VaultRef = { id: one.id, name: one.name }
  void runInvocation(
    createInvocation('openVault', { ...props.getTarget(), vault }),
    props.doing,
    ANSWER_WORDS,
  )
}

function onKeyDown(event: KeyboardEvent) {
  if (event.defaultPrevented || !welcoming.value) return
  const pos = getVaultForKey(event, onList.value.length)
  const one = pos === null ? undefined : onList.value[pos]
  if (!one) return
  event.preventDefault()
  onOpen(one.id)
}

onMounted(() => globalThis.addEventListener('keydown', onKeyDown))
onUnmounted(() => globalThis.removeEventListener('keydown', onKeyDown))
</script>

<template>
  <WelcomePage
    :ways="ways"
    :vaults="onList"
    :heading="words.vaults"
    :offer="adding"
    :version="VERSION"
    @run="onRun"
    @open="onOpen"
    @take-offer="runCommand('newVault', getTarget())"
  />
</template>
