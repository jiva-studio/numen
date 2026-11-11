<script setup lang="ts">
/**
 * The screen a window holding nothing shows: the vaults this installation
 * holds, and the ways into the one it is showing.
 *
 * chosen here, where the window's own icons are. A letter alone opens the vault
 * standing at it, which is the letter drawn on that row.
 */
import { computed, onMounted, onUnmounted } from 'vue'
import { getVaultForKey, WelcomePage } from '@numen/workbench'
import { runInBackground } from '@numen/ui'
import { runInvocation, createInvocation } from '@/features/command-palette'
import { COMMAND, createRowKeys, findIcon, type VaultRef } from '@/entities/command'
import { formatError } from '@/shared/api/errors'
import { VERSION } from '../lib/version'
import { COMMANDS, createVaultRows, createWelcomeActions } from '../lib/screen'
import { WORDS as words } from '@/shared/ui/words'
import { WELCOME_WORDS } from '../words'
import type { WelcomeScreenProps } from './welcomeScreen.types'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<WelcomeScreenProps>(), {
  agent: () => navigator.userAgent,
})

/* --------------------------------- State ---------------------------------- */
const adding = computed(() => {
  const icon = findIcon(COMMAND.createVault)
  return {
    text: words.newVault,
    detail: WELCOME_WORDS.newVaultDetail,
    ...(icon ? { icon } : {}),
    ...createRowKeys(COMMAND.createVault, props.agent),
  }
})

const ways = computed(() => {
  const target = props.getTarget()
  return createWelcomeActions(
    { vault: target.vault.id, isReady: target.isReady },
    { ...words, ...WELCOME_WORDS },
    props.agent,
    props.pluginWays,
  ).map((one) => {
    const icon = one.icon ?? findIcon(one.id)
    return { ...one, ...(icon ? { icon } : {}) }
  })
})

const onList = computed(() => createVaultRows(props.vaultList, { ...words, ...WELCOME_WORDS }))

/** Whether the welcome screen is the one the user is viewing and typing into. */
const welcoming = computed(
  () => props.tabs.length === 0 && !props.search.open.value && !props.commands.palette.open.value,
)

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => globalThis.addEventListener('keydown', onKeyDown))
onUnmounted(() => globalThis.removeEventListener('keydown', onKeyDown))

/* -------------------------------- Handlers -------------------------------- */
function onRun(id: string) {
  if (id !== COMMANDS) return props.runCommand(id, props.getTarget())
  props.search.setOpen(false)
  props.commands.palette.setOpen(true)
}

function onOpen(id: string) {
  const one = props.vaultList.vaults.find((vault) => vault.id === id)
  if (!one) return
  if (id === props.vaultList.showing) {
    props.openCurrentVault?.()
    return
  }
  const vault: VaultRef = { id: one.id, name: one.name }
  runInBackground(
    runInvocation(
      createInvocation(COMMAND.openVault, { ...props.getTarget(), vault }),
      props.doing,
    ),
    (error) => props.doing.writeMessage(formatError(error), 'error'),
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
</script>

<template>
  <WelcomePage
    :ways="ways"
    :vaults="onList"
    :heading="words.vaults"
    :footer="adding"
    :version="VERSION"
    @run="onRun"
    @open="onOpen"
    @take-footer="runCommand(COMMAND.createVault, getTarget())"
  />
</template>
