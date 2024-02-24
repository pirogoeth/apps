id           = "nomad-event-stream_redis-data-1"
name         = "nomad-event-stream_redis-data-1"
type         = "csi"
plugin_id    = "truenas"
capacity_max = "16G"
capacity_min = "8G"

capability {
  access_mode     = "single-node-reader-only"
  attachment_mode = "file-system"
}

capability {
  access_mode     = "single-node-writer"
  attachment_mode = "file-system"
}

mount_options {
  fs_type     = "nfs"
}