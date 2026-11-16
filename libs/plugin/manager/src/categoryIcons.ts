import {
  Bot,
  BookOpen,
  Cpu,
  PenLine,
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
  engines: Cpu,
  readers: BookOpen,
  themes: Palette,
  tools: Wrench,
  views: LayoutPanelLeft,
  system: Settings,
  appearance: SunMoon,
  editor: PenLine,
}

/** The icon a row of the category, or of the System, Appearance and Editor pages, is drawn with. */
export const getCategoryIcon = (category: string): Component => ICONS[category] ?? Puzzle
