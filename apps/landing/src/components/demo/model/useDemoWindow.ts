import { onBeforeUnmount, ref } from 'vue'
import { closeTab } from '@numen/workbench'
import { DEMO_TABS, createDemoWorkspace } from '../lib/tabs'
import { createAgent } from './agent'
import { createPlayer } from './player'

/**
 * The state of the demo window: its agent conversation, its player, and the workspace of panes and
 * tabs. The player pauses when the window unmounts.
 */
export function useDemoWindow() {
  const agent = createAgent()
  const player = createPlayer()
  const workspace = ref(createDemoWorkspace())
  const tabs = ref(DEMO_TABS)

  onBeforeUnmount(player.pause)

  function close(id: string): void {
    workspace.value = closeTab(workspace.value, id)
    tabs.value = tabs.value.filter((one) => one.id !== id)
  }

  const getKind = (id: string) => tabs.value.find((one) => one.id === id)?.kind

  return { agent, player, workspace, tabs, close, getKind }
}
