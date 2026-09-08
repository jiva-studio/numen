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
import { opensVault, WelcomePage } from '@numen/ui'
import type { Tab } from '@numen/ui'
import { invocationOf, type CommandTarget, type VaultRef } from '../command/target'
import type { Commands } from '../command/palette'
import type { VaultList } from '../core'
import type { CommandDeps } from '../command/deps'
import { does } from '../command/handlers'
import type { SearchState } from '../command/search'
import { iconFor } from '../icons'
import { keysOf } from '../command/chords'
import { VERSION } from '../version'
import { COMMANDS, vaultsOn, waysIn } from './screen'
import { WORDS as words } from '../words'

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
  carries: (id: string, at: CommandTarget) => void
}>()

/** What the welcome screen offers below the list of vaults. */
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

/**
 * A way in taken on the welcome screen. The commands are the window's own; the
 * rest are commands, and one that is refused says why.
 */
const runs = (id: string) => {
  if (id !== COMMANDS) return props.carries(id, props.where())
  props.search.shows(false)
  props.commands.shows(true)
}

/** Whether the welcome screen is what the person is looking at and typing into. */
const welcoming = computed(
  () => props.tabs.length === 0 && !props.search.open.value && !props.commands.open.value,
)

/** A vault chosen on the welcome screen, shown in this window in place of none. */
const opens = (id: string) => {
  const one = props.listed.vaults.find((vault) => vault.id === id)
  if (!one) return
  const vault: VaultRef = { id: one.id, name: one.name }
  void does(invocationOf('openVault', { ...props.where(), vault }), props.doing, words)
}

/** A letter alone, which opens the vault drawn on the row carrying it. */
const asked = (event: KeyboardEvent) => {
  if (event.defaultPrevented || !welcoming.value) return
  const at = opensVault(event, onList.value.length)
  const one = at === null ? undefined : onList.value[at]
  if (!one) return
  event.preventDefault()
  opens(one.id)
}

onMounted(() => globalThis.addEventListener('keydown', asked))
onUnmounted(() => globalThis.removeEventListener('keydown', asked))
</script>

<template>
  <WelcomePage
    :ways="ways"
    :vaults="onList"
    :heading="words.vaults"
    :offer="adding"
    :version="VERSION"
    @runs="runs"
    @opens="opens"
    @offers="carries('newVault', where())"
  />
</template>
