import { findPane, type Workspace } from '@numen/workbench'
import type { JsonObject, JsonValue } from '@/shared/api/json'
import type { Result } from '@numen/result'

const isObject = (value: JsonValue | undefined): value is JsonObject =>
  typeof value === 'object' && value !== null && !Array.isArray(value)

function hasPaneTabs(node: JsonObject, remaining: Set<string>, isRoot: boolean): boolean {
  const { tabs, active } = node
  if (!Array.isArray(tabs)) return false
  if (tabs.length === 0) return active === null && isRoot
  return (
    tabs.includes(active) &&
    tabs.every((tab: JsonValue) => typeof tab === 'string' && remaining.delete(tab))
  )
}

function hasValidSizes(sizes: JsonValue | undefined, count: number): boolean {
  if (!Array.isArray(sizes) || sizes.length !== count) return false
  let total = 0
  for (const size of sizes) {
    if (typeof size !== 'number' || !Number.isFinite(size) || size <= 0) return false
    total += size
  }
  return Math.abs(total - 1) <= Number.EPSILON * count
}

function isWorkspaceNode(
  value: JsonValue | undefined,
  nodes: Set<string>,
  remaining: Set<string>,
): value is JsonObject & Workspace['root'] {
  if (!isObject(value) || typeof value.id !== 'string' || !value.id || nodes.has(value.id))
    return false
  nodes.add(value.id)
  if (value.kind === 'pane') return hasPaneTabs(value, remaining, nodes.size === 1)
  const { children, sizes } = value
  return (
    value.kind === 'branch' &&
    Array.isArray(children) &&
    children.length >= 2 &&
    hasValidSizes(sizes, children.length) &&
    children.every((child: JsonValue) => isWorkspaceNode(child, nodes, remaining))
  )
}

/** Reads a workspace whose panes contain each saved tab exactly once. */
export function readSavedLayout(
  json: string,
  tabs: readonly string[],
): Result<Workspace, 'invalid-layout'> {
  try {
    const value: JsonValue = JSON.parse(json)
    const remaining = new Set(tabs)
    if (
      isObject(value) &&
      (value.axis === 'horizontal' || value.axis === 'vertical') &&
      typeof value.focus === 'string' &&
      isWorkspaceNode(value.root, new Set(), remaining) &&
      remaining.size === 0 &&
      findPane(value.root, value.focus)
    ) {
      return { ok: true, value: { root: value.root, axis: value.axis, focus: value.focus } }
    }
  } catch {
    return { ok: false, error: 'invalid-layout' }
  }
  return { ok: false, error: 'invalid-layout' }
}
