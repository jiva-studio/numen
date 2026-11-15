/** The keys of the person: each has a name and is read from an environment variable of the machine, so no key is written in a scenario or a report. */
export class Keys {
  private readonly variables = new Map<string, string>()

  declare(table: readonly (readonly string[])[]): void {
    const [head = [], ...rows] = table
    const nameColumn = head.indexOf('key name')
    const variableColumn = head.indexOf('environment variable')
    if (nameColumn < 0 || variableColumn < 0)
      throw new Error('the table has no "key name" or no "environment variable": give both columns')
    for (const row of rows) {
      const name = (row[nameColumn] ?? '').trim()
      const variable = (row[variableColumn] ?? '').trim()
      if (name === '' || variable === '')
        throw new Error('a row of keys has no "key name" or no "environment variable": fill both')
      if ((process.env[variable] ?? '') === '')
        throw new Error(`the environment variable ${variable} for the key "${name}" is not set or is empty: set it before the run`)
      this.variables.set(name, variable)
    }
  }

  /** The key a declared name stands for, and the text itself for any other text. */
  read(text: string): string {
    const variable = this.variables.get(text)
    return variable === undefined ? text : (process.env[variable] ?? '')
  }
}
