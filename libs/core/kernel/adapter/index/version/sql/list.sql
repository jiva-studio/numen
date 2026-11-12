SELECT path, plugin_id, key, source_version_id, source_property_key, source_property_value, kind, file, input_hash, text_hash, create_time
FROM text_versions
WHERE vault = ?
ORDER BY id;
