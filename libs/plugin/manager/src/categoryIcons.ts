import {
  Bot,
  BookOpen,
  Cpu,
  KeyRound,
  LayoutPanelLeft,
  Palette,
  Plug,
  Puzzle,
  Settings,
  Wrench,
} from '@lucide/vue'
import type { Component } from 'vue'

const ICONS: Readonly<Record<string, Component>> = {
  agents: Bot,
  connections: Plug,
  engines: Cpu,
  readers: BookOpen,
  themes: Palette,
  tools: Wrench,
  views: LayoutPanelLeft,
  system: Settings,
  secrets: KeyRound,
}

/** The icon a row of the category, or of the System and Secrets pages, is drawn with. */
export const getCategoryIcon = (category: string): Component => ICONS[category] ?? Puzzle
