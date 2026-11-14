# Templates

Makes a new note from a template: a note in the templates folder, with its variables filled in. The user guide is [docs/templates.md](../../docs/templates.md).

The frontend lists the templates and runs the command `note.createFromTemplate`. The backend reads the template through the host, replaces its variables and creates the note with its own `id`. The setting `folder` names the templates folder (`Templates`).
