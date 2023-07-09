# go-xdcc
Use the XDCC protocol to download, organize, and track anime using an IRC client.


## 🪛 Installation
Ensure that the requsite directories exist and are writable by the user running the application.

```bash
mkdir xdcc && cd xdcc
mkdir -p config/trackers
mkdir downloads

curl https://raw.githubusercontent.com/Elfshot/go-xdcc/main/config/config.yaml.example -o config/config.yaml -s

chown 1000:1000 -R config && chown 1000:1000 downloads
chmod 766 -R config && chmod 766 downloads
```
> Note: The above commands assume that the application is running as user 1000:1000.

> ⚠️ An appropriate config.yaml file must be created in the config directory.

## ⚙️ Example config.yaml
```yaml
maxDownloads: 3 # Max number of downloads at once- 3 is the recommended max
preferedBots: ["Ghouls|Arutha", "ARUTHA-BATCH|1080p", "ARUTHA-BATCH|720p", "CR-HOLLAND|NEW", "CR-ARUTHA|NEW"] # List of bots to prefer | This order is the order in which packs will be sourced from
preferedFormat: "1080p" # 1080p, 720p, 480p
downloadDir: "./downloads" # (Leave as-is for docker instances) Directory to download to
#boundIp: "000.000.000.000" # (Optional [auto-binds to first available IP]) IP to bind to for DCC TCP connections
downloadInterval: 30 # Time in minutes between searching for new downloads

irc:
  server: "irc.rizon.net"
  serverPort: 6697
  channelName: "#nibl"
  closeConnectionMins: 5 # Time in minutes of an idle irc connection before closing it
  maxWaitIrcCycles: 25 # Max number of irc cycles to wait for a download to start before erroring
```
[NIBL Bots](https://nibl.co.uk/bots) should be used to find the prefered bots. The names should be copied over exactly.

## 🔍 Example tracker config
Tracker config files must be placed in the config/trackers directory.

```yaml
searchName: Dr. Stone
fileName: Dr. Stone
season: 1
episodeRange: [ 1,24 ]
```

| Parameter | Usage |
| ------- | ------- |
| searchName | The name to search for. Should be as appears on NIBL |
| fileName | The name to use for the downloaded anime's folder |
| season | The season number |
| episodeRange | The range of episodes to download. These values are inclusive |

[TheTVDB](https://thetvdb.com/) should be used to determine the season number, episode ranges, and file name.

[NIBL](https://nibl.co.uk/) should be used to determine the search name.

## 🐳 Example docker-compose.yaml
```yaml
version: "3.5"
services:
  goxdcc:
    image: "ghcr.io/elfshot/goxdcc:main"
    working_dir: /xdcc
    container_name: "goxdcc"
    # Use network mode to attach a VPN or proxy container as DDC connections are not encrypted
    # network_mode: "container:<container hosting network name or id | No arrows>"
    user: 1000:1000
    resources:
      limits: # Absolute limits
        cpus: '1.5' # 1.5 cores
        memory: 1G
      reservations: # Minimum resources
        memory: 100M
    volumes:
      - pathToLocalConfig:/xdcc/config:rw
      - pathToLocalDownloads:/xdcc/downloads:rw
    restart: unless-stopped
    environment:
      - LOG_LEVEL=INFO #DEBUG, INFO, ERROR (Default)
    tty: true
    stdin_open: true
```