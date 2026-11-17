import type { Table } from './table'

/** How many columns the table has, counting the widest row. */
export const getWidth = (table: Table): number =>
  Math.max(table.align.length, table.head.length, ...table.body.map((row) => row.length), 1)
