import {
  Bot,
  BookOpen,
  Cpu,
  KeyRound,
  Laptop,
  PenLine,
  Search,
  LayoutPanelLeft,
  Palette,
  Plug,
  Puzzle,
  Settings,
  SunMoon,
  Wrench,
} from '@lucide/vue'
import type { Component } from 'vue'

const ICONS: Readonly<Record<string, Component>> = {
  agents: Bot,
  connections: Plug,
  device: Laptop,
  engines: Cpu,
  identity: KeyRound,
  readers: BookOpen,
  themes: Palette,
  tools: Wrench,
  views: LayoutPanelLeft,
  system: Settings,
  appearance: SunMoon,
  editor: PenLine,
  search: Search,
}

/** The icon a row of the category, or of the System, Appearance, Editor and Search pages, is drawn with. */
export const getCategoryIcon = (category: string): Component => ICONS[category] ?? Puzzle
