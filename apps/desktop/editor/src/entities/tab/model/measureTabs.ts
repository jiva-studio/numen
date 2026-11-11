import { isMeasurable } from '../lib/measure'
import type { WindowTabs } from './windowTabs'

/** Lays the content of every open tab that can be measured out again, as after a change of text size. */
export function measureTabs(windowTabs: Pick<WindowTabs, 'tabs'>): void {
  for (const tab of windowTabs.tabs.tabs.value) {
    const state = windowTabs.tabs.getTab(tab.id)?.state
    if (state && isMeasurable(state)) state.measure()
  }
}
