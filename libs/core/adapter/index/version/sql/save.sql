-- A version recorded again keeps its row, and with it its place in the order.
INSERT INTO text_versions
    (vault, path, plugin_id, key, source_version_id, source_property_key, source_property_value, kind, file, source_hash, hash, create_time)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (vault, path, plugin_id, key) DO UPDATE SET
    source_version_id = excluded.source_version_id,
    source_property_key = excluded.source_property_key,
    source_property_value = excluded.source_property_value,
    kind = excluded.kind,
    file = excluded.file,
    source_hash = excluded.source_hash,
    hash = excluded.hash,
    create_time = excluded.create_time;
