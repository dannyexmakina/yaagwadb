package assets

import "embed"

const (
	// ConfigFileName is the example config file. install places it if no config file exists.
	ConfigFileName = "config.toml"
	// DaemonUnitFileName, MountUnitFileName and HranaUnitFileName are systemd units. install places them in the systemd unit directory.
	DaemonUnitFileName = "yaagwadb.service"
	MountUnitFileName  = "yaagwadb-mount.service"
	HranaUnitFileName  = "yaagwadb-hrana.service"
)

//go:embed config.toml yaagwadb.service yaagwadb-mount.service yaagwadb-hrana.service
var Files embed.FS
