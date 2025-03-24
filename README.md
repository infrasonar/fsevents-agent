[![CI](https://github.com/infrasonar/fsevents-agent/workflows/CI/badge.svg)](https://github.com/infrasonar/fsevents-agent/actions)
[![Release Version](https://img.shields.io/github/release/infrasonar/fsevents-agent)](https://github.com/infrasonar/fsevents-agent/releases)

# InfraSonar FileSystem Events Agent

Documentation: https://docs.infrasonar.com/collectors/agents/fsevents/

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
`WATCH_PATHS`               | `watch.cnf`                   | Configuration file with paths to watch for file system events.
`FN_STATE_JSON`             | `state.json`                  | File where the state is stored.
`CACHE_BPS_THRESHOLD`       | `800`                         | Threshold in MB/s which defines when the file is from tape or cache.

## Build
```
CGO_ENABLED=0 go build -trimpath -o fsevents-agent
```


## Installation

Download the latest release:
```bash
wget https://github.com/infrasonar/fsevents-agent/releases/download/v0.1.0/fsevents-agent
```

> _The pre-build binary is build for the **fsevents-amd64** platform. For other platforms build from source using the command:_ `CGO_ENABLED=0 go build -o fsevents-agent`

Ensure the binary is executable:
```
chmod +x fsevents-agent
```

Copy the binary to `/usr/sbin/infrasonar-fsevents-agent`

```
sudo cp fsevents-agent /usr/sbin/infrasonar-fsevents-agent
```

### Using Systemd

```bash
sudo touch /etc/systemd/system/infrasonar-fsevents-agent.service
sudo chmod 664 /etc/systemd/system/infrasonar-fsevents-agent.service
```

**1. Using you favorite editor, add the content below to the file created:**

```
[Unit]
Description=InfraSonar fsevents Agent
Wants=network.target

[Service]
EnvironmentFile=/etc/infrasonar/fsevents-agent.env
ExecStart=/usr/sbin/infrasonar-fsevents-agent

[Install]
WantedBy=multi-user.target
```

**2. Create the directory `/etc/infrasonar`**

```bash
sudo mkdir /etc/infrasonar
```

**3. Create the file `/etc/infrasonar/fsevents-agent.env` with at least:**

```
TOKEN=<YOUR TOKEN HERE>
```

Optionaly, add environment variable to the `fsevents-agent.env` file for settings like `ASSET_ID` or `CONFIG_PATH` _(see all [environment variables](#environment-variables) in the table above)_.

**4. Reload systemd:**

```bash
sudo systemctl daemon-reload
```

**5. Install the service:**

```bash
sudo systemctl enable infrasonar-fsevents-agent
```

**Finally, you may want to start/stop or view the status:**
```bash
sudo systemctl start infrasonar-fsevents-agent
sudo systemctl stop infrasonar-fsevents-agent
sudo systemctl status infrasonar-fsevents-agent
```

**View logging:**
```bash
journalctl -u infrasonar-fsevents-agent
```

# fseventss-agent
