[![CI](https://github.com/infrasonar/fsevent-agent/workflows/CI/badge.svg)](https://github.com/infrasonar/fsevent-agent/actions)
[![Release Version](https://img.shields.io/github/release/infrasonar/fsevent-agent)](https://github.com/infrasonar/fsevent-agent/releases)

# InfraSonar FSevent Agent

Documentation: https://docs.infrasonar.com/collectors/agents/fsevent/

## Environment variables

Environment                 | Default                       | Description
----------------------------|-------------------------------|-------------------
`CONFIG_PATH`       		| `/etc/infrasonar` 			| Path where configuration files are loaded and stored _(note: for a user, the `$HOME` path will be used instead of `/etc`)_
`TOKEN`                     | _required_                    | Token used for authentication _(This MUST be a container token)_.
`ASSET_NAME`                | _none_                        | Initial Asset Name. This will only be used at the announce. Once the asset is created, `ASSET_NAME` will be ignored.
`ASSET_ID`                  | _none_                        | Asset Id _(If not given, the asset Id will be stored and loaded from file)_.
`API_URI`                   | https://api.infrasonar.com    | InfraSonar API.
`SKIP_VERIFY`               | _none_                        | Set to `1` or something else to skip certificate validation.
`CHECK_FS`                  | `300`                         | Interval in seconds for the `fs` check.


## Build
```
CGO_ENABLED=0 go build -trimpath -o fsevent-agent
```

Or, solaris build:
```
GOOS=solaris GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -o fsevent-agent.solaris-amd64
```

## Installation

Download the latest release:
```bash
wget https://github.com/infrasonar/fsevent-agent/releases/download/v0.1.0/fsevent-agent
```

> _The pre-build binary is build for the **fsevent-amd64** platform. For other platforms build from source using the command:_ `CGO_ENABLED=0 go build -o fsevent-agent`

Ensure the binary is executable:
```
chmod +x fsevent-agent
```

Copy the binary to `/usr/sbin/infrasonar-fsevent-agent`

```
sudo cp fsevent-agent /usr/sbin/infrasonar-fsevent-agent
```

### Using Systemd

```bash
sudo touch /etc/systemd/system/infrasonar-fsevent-agent.service
sudo chmod 664 /etc/systemd/system/infrasonar-fsevent-agent.service
```

**1. Using you favorite editor, add the content below to the file created:**

```
[Unit]
Description=InfraSonar fsevent Agent
Wants=network.target

[Service]
EnvironmentFile=/etc/infrasonar/fsevent-agent.env
ExecStart=/usr/sbin/infrasonar-fsevent-agent

[Install]
WantedBy=multi-user.target
```

**2. Create the directory `/etc/infrasonar`**

```bash
sudo mkdir /etc/infrasonar
```

**3. Create the file `/etc/infrasonar/fsevent-agent.env` with at least:**

```
TOKEN=<YOUR TOKEN HERE>
```

Optionaly, add environment variable to the `fsevent-agent.env` file for settings like `ASSET_ID` or `CONFIG_PATH` _(see all [environment variables](#environment-variables) in the table above)_.

**4. Reload systemd:**

```bash
sudo systemctl daemon-reload
```

**5. Install the service:**

```bash
sudo systemctl enable infrasonar-fsevent-agent
```

**Finally, you may want to start/stop or view the status:**
```bash
sudo systemctl start infrasonar-fsevent-agent
sudo systemctl stop infrasonar-fsevent-agent
sudo systemctl status infrasonar-fsevent-agent
```

**View logging:**
```bash
journalctl -u infrasonar-fsevent-agent
```

# fsevents-agent
