/**
 * The state and handlers of the welcome screen: the vault rows, the ways into the vault shown,
 * the shortcut keys on the window, and the command invocations that open a vault or run a way.
 */
import { computed, onMounted, onUnmounted } from 'vue'
import { getVaultForKey, type Tab, type WelcomeAction } from '@numen/workbench'
import { runInBackground, win } from '@numen/ui'
import {
  createInvocation,
  runInvocation,
  type Commands,
  type SearchState,
} from '@/features/command-palette'
import {
  COMMAND,
  createRowKeys,
  findIcon,
  type CommandDeps,
  type CommandTarget,
  type VaultRef,
} from '@/shared/model/command'
import type { VaultList } from '@/entities/vault'
import { formatError } from '@/shared/api/errors'
import { WORDS as words } from '@/shared/ui/words'
import { COMMANDS, createVaultRows, createWelcomeActions } from '../lib/screen'
import { WELCOME_WORDS } from '../words'

export interface WelcomeScreenProps {
  vaultList: VaultList
  tabs: readonly Tab[]
  commands: Commands
  search: SearchState
  doing: CommandDeps
  getTarget: () => CommandTarget
  pluginWays: readonly WelcomeAction[]
  runCommand: (id: string, target: CommandTarget) => void
  agent: string
  openCurrentVault?: (() => void) | undefined
}

export function useWelcomeScreen(props: WelcomeScreenProps) {
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

  const vaults = computed(() => createVaultRows(props.vaultList, { ...words, ...WELCOME_WORDS }))

  /** Whether the welcome screen is the one the user is viewing and typing into. */
  const isWelcoming = computed(
    () => props.tabs.length === 0 && !props.search.open.value && !props.commands.palette.open.value,
  )

  function onRun(id: string): void {
    if (id !== COMMANDS) {
      props.runCommand(id, props.getTarget())
      return
    }
    props.search.setOpen(false)
    props.commands.palette.setOpen(true)
  }

  function onOpen(id: string): void {
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

  function onTakeFooter(): void {
    props.runCommand(COMMAND.createVault, props.getTarget())
  }

  function onKeyDown(event: KeyboardEvent): void {
    if (event.defaultPrevented || !isWelcoming.value) return
    const pos = getVaultForKey(event, vaults.value.length)
    const one = pos === null ? undefined : vaults.value[pos]
    if (!one) return
    event.preventDefault()
    onOpen(one.id)
  }

  onMounted(() => win.addEventListener('keydown', onKeyDown))
  onUnmounted(() => win.removeEventListener('keydown', onKeyDown))

  return {
    adding,
    ways,
    vaults,
    onRun,
    onOpen,
    onTakeFooter,
  }
}
