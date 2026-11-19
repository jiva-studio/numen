# Logs (`md.numen.tools.logs`)

Structured application log viewer for Numen.

## What it shows

- **Live stream & recent history**: Follows `host.work.logs.watch`, displaying recent entries from today's daily JSONL log file (`logs/numen-YYYY-MM-DD.jsonl`) and incoming entries from the core, SQLite pools (`sql`), ConnectRPC handlers (`rpc`), windows (`window`), and plugins.
- **Top filter bar**: Filter entries by severity level (`All`, `Debug`, `Info`, `Warning`, `Error`) and search by message text.
- **Compact summary row**: Shows the timestamp, level badge, source, required `message`, and property key chips (`keys`).
- **Expandable details**: Clicking an entry expands its properties inline. SQL queries (`sql`) are formatted across clauses, and JSON values (`request`, `response`, `args`, and structured objects) are formatted as indented JSON.
