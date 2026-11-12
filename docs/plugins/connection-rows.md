# Rows of connections

A plugin that needs a model service declares a setting of type `connections`. The person fills it on the page of the plugin with one row for each connection to try, and the core reads the rows when the plugin asks for the operation. The plugin never sees a key, an address or the fields of a connection.

## Declare the setting

```json
{
  "key": "connections",
  "type": "connections",
  "title": "Connections",
  "description": "Services tried in order.",
  "operation": "complete",
  "max": 5
}
```

`operation` is the operation the plugin asks for, a lowercase word such as `complete`, `chat` or `embed`. `max` is the most rows the setting holds, a whole number of at least 1. The setting takes `when` as any setting does, so a plugin can show the rows only while another setting holds a given value. A plugin declares at most one `connections` setting for each operation, and the manifest check rejects a setting that declares a `default`, `options`, `optionsBy`, `optionsFrom`, `min`, `step`, `accept`, `multiple`, `preview` or `generate`, or an `operation` on a setting of another type. A plugin that has such a setting uses connections without a permission: the rows are the grant, and the core takes the connections only from the rows of the plugin that calls.

## The stored value

The value is a list of rows in the object of the plugin in `numen.json`, in the order of use:

```json
"md.numen.tools.proofreading": {
  "connections": [
    { "connection": "c1f0a9e20", "model": "google/gemini-2.5-flash", "reasoning": "low" },
    { "connection": "c77b01d40", "model": "qwen2.5:14b" }
  ]
}
```

A row has `connection`, the id of an existing connection, `model`, a name that is not empty, and optionally `reasoning`, a level of the model; an empty or absent `reasoning` asks for the default of the provider. A write is refused when the value is no list, holds more than `max` rows, holds a row with another key, a missing or empty `connection` or `model`, or a connection that does not exist, or holds the same connection and model twice. On read, each row that fails one of these rules, and each row beyond `max`, is left out of force with a notice; the file is not changed.

Deleting a connection removes the rows that name it from every setting of rows, through the same write as any setting, and the Connections page names the plugins whose rows it removes. Editing a connection sends the configuration again to every running plugin whose rows name it, so a plugin that keeps the profile of a model reads it again.

## What the plugin receives

`Configure` carries the rows as the JSON text of the list under the key of the setting, for a setting that is shown. The Go SDK reads them with `connection.ReadRows(values, key)`, which returns `[]connection.Row` with `Connection`, `Model` and `Reasoning`; the TypeScript SDK reads the value of `getSetting(key)` with `readConnectionRows(value)`. A plugin reads its rows only to show or to name a model. The window is the only writer of the rows: `Configure` is read-only for the plugin.

## The page of the plugin

The page shows one row for each connection, with a grip to move it, the connection, the model, the reasoning level and a button that removes it. "Add connection" adds a row until the setting holds `max` rows. Only existing connections can be chosen. A connection whose type is not installed, cannot do the operation, or is missing a required value is dimmed with the reason and listed after the others. The models, their titles and the reasoning levels are asked from the provider of the chosen connection; the model field also takes a name that is typed, and choosing a connection fills it with the first model of the list, except for embeddings, where the person chooses the model and the question about indexing follows once the row is complete; cancelling that question keeps the row with its connection. Under the rows the page says that the text of the notes is sent to each connection listed, in this order; where the setting holds one row, it says the text is sent to the connection chosen there. With no row the page offers "Open Connections", which shows the Connections page; a plugin with no row is off.

A row is moved by dragging its grip. With the keyboard, focus the grip, press Space to pick the row up, press the arrow keys to move it and press Space to drop it; Escape puts it back.
