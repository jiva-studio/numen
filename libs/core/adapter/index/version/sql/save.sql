-- A version recorded again keeps its row, and with it its place in the order.
INSERT INTO text_versions
    (vault, path, plugin_id, key, source_version_id, source_property_key, source_property_value, kind, file, input_hash, text_hash, create_time)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (vault, path, plugin_id, key) DO UPDATE SET
    source_version_id = excluded.source_version_id,
    source_property_key = excluded.source_property_key,
    source_property_value = excluded.source_property_value,
    kind = excluded.kind,
    file = excluded.file,
    input_hash = excluded.input_hash,
    text_hash = excluded.text_hash,
    create_time = excluded.create_time;
