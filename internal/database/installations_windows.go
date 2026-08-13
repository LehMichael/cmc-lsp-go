//go:build windows

package database

import (
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const installedCMCRegistryPath = `SOFTWARE\Siemens\Automation\_InstalledSW`

func installedDatabaseCandidates() []string {
	var installations []cmcInstallation
	for _, registryView := range []uint32{registry.WOW64_64KEY, registry.WOW64_32KEY} {
		installations = append(installations, installedCMCInstallations(registryView)...)
	}
	return databaseCandidatesForInstallations(installations)
}

func installedCMCInstallations(registryView uint32) []cmcInstallation {
	installedSoftware, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		installedCMCRegistryPath,
		registry.ENUMERATE_SUB_KEYS|registryView,
	)
	if err != nil {
		return nil
	}
	defer installedSoftware.Close()

	productKeys, err := installedSoftware.ReadSubKeyNames(-1)
	if err != nil {
		return nil
	}

	var installations []cmcInstallation
	for _, productKey := range productKeys {
		component, err := registry.OpenKey(
			installedSoftware,
			filepath.Join(productKey, "SIEMENS_CMC_DIFF"),
			registry.QUERY_VALUE|registryView,
		)
		if err != nil {
			continue
		}
		path, _, pathErr := component.GetStringValue("Path")
		version, _, versionErr := component.GetStringValue("Version")
		if versionErr != nil {
			version, _, versionErr = component.GetStringValue("Release")
		}
		component.Close()
		if pathErr == nil && versionErr == nil {
			installations = append(installations, cmcInstallation{version: version, path: path})
		}
	}
	return installations
}
