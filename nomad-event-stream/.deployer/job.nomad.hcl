variable "version" {
    description = "App deployment version"
}

job "nomad-event-stream" {
  type = "service"
  namespace = "nomad-system"
  datacenters = ["dc1"]

  node_pool = "default"

  group "app" {
    count = 1

    update {
      max_parallel = 1
      min_healthy_time = "30s"
      healthy_deadline = "5m"
      auto_revert = true
      auto_promote = false
    }

    network {
      port "http" {
        to = 8000
      }
    }

    task "nomad-event-stream" {
      driver = "docker"

      config {
        image = "ghcr.io/pirogoeth/apps/nomad-event-stream:${var.version}"
        force_pull = true
        ports = ["http"]

        volumes = [
          "/opt/nomad/tls:/opt/nomad/tls:ro",
        ]

        labels = {
          appname = "nomad-event-stream"
          component = "app"
          vector_stdout_parse_mode = "json"
        }
      }

      template {
        destination = "local/env"
        change_mode = "restart"
        env         = true

        data = <<EOH
NOMAD_ADDR=https://10.100.10.32:4646
NOMAD_CACERT=/opt/nomad/tls/nomad-agent-ca.pem
NOMAD_CERT=/opt/nomad/tls/global-cli-nomad.pem
NOMAD_KEY=/opt/nomad/tls/global-cli-nomad-key.pem
NOMAD_SKIP_VERIFY=true

REDIS_STREAM=nomad:events
{{range nomadService "nomad-event-stream-redis"}}
REDIS_URL=redis://{{.Address}}:{{.Port}}/0
{{end}}
EOH
      }

      resources {
        cpu    = 256
        memory = 256
      }
    }
  }

  group "redis" {
    count = 1

    network {
      port "redis" {
        to = 6379
      }
    }

    volume "data" {
      type            = "csi"
      source          = "nomad-event-stream_redis-data-1"
      read_only       = false
      attachment_mode = "file-system"
      access_mode     = "single-node-writer"
    }

    task "redis" {
      driver = "docker"

      config {
        image = "redis:latest"
        force_pull = true
        ports = ["redis"]

        labels = {
          appname = "nomad-event-stream"
          component = "redis"
          vector_stdout_parse_mode = "plain"
        }
      }

      volume_mount {
        volume      = "data"
        destination = "/data"
      }

      template {
        change_mode = "signal"
        change_signal = "SIGHUP"

        destination = "/usr/local/etc/redis/redis.conf"
        data = <<EOF
port 6379
dir /data/rdb
appendonlydir /data/aof
appendonly yes
save 3600 1 300 100 60 10000
stop-writes-on-bgsave-error yes
rdbcompression yes
rdbchecksum yes
always-show-logo no
EOF
      }

      resources {
        cpu    = 256
        memory = 256
      }

      service {
        name = "nomad-event-stream-redis"
        port = "redis"
        provider = "nomad"
      }
    }
  }
}
