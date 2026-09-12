<script setup lang="ts">
/**
 * The screen a window holding nothing shows: the vaults this installation
 * holds, and the ways into the one it is showing.
 *
 * The screen draws what it is given, so what stands in front of each way is
 * chosen here, where the window's own icons are. A letter alone opens the vault
 * standing at it, which is the letter drawn on that row.
 */
import { computed, onMounted, onUnmounted } from 'vue'
import { getVaultForKey, WelcomePage } from '@numen/ui'
import type { Tab } from '@numen/ui'
import {
  runInvocation,
  invocationOf,
  keysOf,
  type CommandDeps,
  type Commands,
  type CommandTarget,
  type SearchState,
  type VaultRef,
} from '@/features/command-palette'
import type { VaultList } from '@/shared/vaults'
import { iconFor } from '@/shared/icons'
import { VERSION } from '../lib/version'
import { COMMANDS, vaultsOn, waysIn } from '../lib/screen'
import { WORDS as words } from '@/shared/words'

// --- Props & Emits ---
const props = defineProps<{
  /** Every vault the installation holds, as the list last answered. */
  listed: VaultList
  /** Every tab the window holds, which is none while this screen is what shows. */
  tabs: readonly Tab[]
  commands: Commands
  search: SearchState
  doing: CommandDeps
  where: () => CommandTarget
  /** A command asked for, which is what every way in but the commands comes to. */
  runCommand: (id: string, at: CommandTarget) => void
}>()

// --- State ---
const adding = computed(() => {
  const icon = iconFor('newVault')
  return {
    text: words.newVault,
    detail: words.newVaultDetail,
    ...(icon ? { icon } : {}),
    ...keysOf('newVault', navigator.userAgent),
  }
})

const ways = computed(() => {
  const at = props.where()
  return waysIn({ vault: at.vault.id, ready: at.ready }, words, navigator.userAgent).map((one) => {
    const icon = iconFor(one.id)
    return { ...one, ...(icon ? { icon } : {}) }
  })
})

const onList = computed(() => vaultsOn(props.listed, words))

/** Whether the welcome screen is what the person is looking at and typing into. */
const welcoming = computed(
  () => props.tabs.length === 0 && !props.search.open.value && !props.commands.open.value,
)

// --- Handlers ---
function onRuns(id: string) {
  if (id !== COMMANDS) return props.runCommand(id, props.where())
  ;props.search.setOpen(false)
  ;props.commands.setOpen(true)
}

function onOpens(id: string) {
  const one = props.listed.vaults.find((vault) => vault.id === id)
  if (!one) return
  const vault: VaultRef = { id: one.id, name: one.name }
  void runInvocation(invocationOf('openVault', { ...props.where(), vault }), props.doing, words)
}

function onKeyDown(event: KeyboardEvent) {
  if (event.defaultPrevented || !welcoming.value) return
  const at = getVaultForKey(event, onList.value.length)
  const one = at === null ? undefined : onList.value[at]
  if (!one) return
  event.preventDefault()
  onOpens(one.id)
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
    @runs="onRuns"
    @opens="onOpens"
    @offers="runCommand('newVault', where())"
  />
</template>
