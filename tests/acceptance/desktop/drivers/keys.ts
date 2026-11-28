/** A key whose variable is not set on this machine: a scenario that needs it cannot run here. */
export class MissingKeyError extends Error {}

/** The keys of the person: each has a name and is read from an environment variable of the machine, so no key is written in a scenario or a report. */
const NAME = 'key name'
const VARIABLE = 'environment variable'

const readRows = (table: readonly (readonly string[])[]): [string, string][] => {
  const [head = [], ...rows] = table
  const nameColumn = head.indexOf(NAME)
  const variableColumn = head.indexOf(VARIABLE)
  if (nameColumn < 0 || variableColumn < 0)
    throw new Error(`the table has no "${NAME}" or no "${VARIABLE}": give both columns`)
  return rows.map((row) => [(row[nameColumn] ?? '').trim(), (row[variableColumn] ?? '').trim()])
}

export class Keys {
  private readonly variables = new Map<string, string>()

  declare(table: readonly (readonly string[])[]): void {
    for (const [name, variable] of readRows(table)) {
      if (name === '' || variable === '')
        throw new Error(`a row of keys has no "${NAME}" or no "${VARIABLE}": fill both`)
      if ((process.env[variable] ?? '') === '')
        throw new MissingKeyError(`the environment variable ${variable} for the key "${name}" is not set or is empty: set it before the run`)
      this.variables.set(name, variable)
    }
  }

  /** The key a declared name stands for, and the text itself for any other text. */
  read(text: string): string {
    const variable = this.variables.get(text)
    return variable === undefined ? text : (process.env[variable] ?? '')
  }
}
