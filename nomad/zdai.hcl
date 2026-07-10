job "zdai" {
  datacenters = ["dc1"]
  type = "service"

  affinity {
    attribute = "${node.unique.name}"
    value = "(a|m|z)[0-9]dune.*"
    operator = "regexp"
    weight = -100
  }

  update {
    max_parallel = 1
    stagger = "20s"
    auto_revert = true
  }

  group "zdai" {
		affinity {
			attribute = "${node.unique.name}"
      value = "zp[0-9]dune.*"
      operator = "regexp"
			weight = 100
		}

    count = 1

    network {
      port "micro" { to = 3001 }
      port "broker" { to = 3002 }

      dns {
        servers = ["10.0.0.6", "1.1.1.1"]
      }
    }

    task "zdai" {
      driver = "podman"

      config {
        image = "reg.zerodoc.dev/zerodoc/zdai:main"
        auth {
          username = "${reg_user}"
          password = "${reg_pass}"
        }
        force_pull = true

        ports = ["micro", "broker"]

        volumes = [
          "/mnt/local/syncthing/data1:/vault:rw",
          "/mnt/local/zdai/state:/state",
          "/mnt/local/zdai/claude:/root/.claude",
        ]
      }

      env {
        ENV="prod"
        APPROLE_ID="${approle_id}"
        APPROLE_SECRET="${approle_secret}"
        GIT_COMMIT="${DRONE_COMMIT}"
        MICRO_PORT=3001
        BROKER_PORT=3002
        VAULT_ADDRESS="${vault_address}"
        VAULT_DIR="/vault"
        STATE_DIR="/state"
        ZDCLAUDE_REPO="https://github.com/ZeroDoctor/zdclaude"
      }
    }

    service {
      name = "zdai"
      port = "micro"

      tags = [
        "traefik.enable=false",
      ]

      check {
        name = "alive"
        type = "tcp"
        interval = "10s"
        timeout  = "2s"
      }
    }
  }
}
