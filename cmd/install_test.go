package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"yaagwadb/assets"
)

const (
	installTestConfigName     = "yaagwadb.toml"
	installTestExisting       = "existing"
	installTestUnitDirectory  = "units"
	installTestContentFormat  = "%s: content differs from the embedded asset"
	installTestKeptFormat     = "existing config was overwritten: %q"
	installTestKeptUnitFormat = "existing unit was overwritten: %q"
	installTestWrittenFormat  = "installConfig wrote = %v, want %v"
)

func TestInstallConfigWritesOnlyWhenMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), installTestConfigName)
	written, err := installConfig(path)
	if err != nil || !written {
		t.Fatalf(installTestWrittenFormat, written, true)
	}
	assertSameAsAsset(t, path, assets.ConfigFileName)

	if err := os.WriteFile(path, []byte(installTestExisting), configFilePermission); err != nil {
		t.Fatal(err)
	}
	written, err = installConfig(path)
	if err != nil || written {
		t.Fatalf(installTestWrittenFormat, written, false)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != installTestExisting {
		t.Fatalf(installTestKeptFormat, data)
	}
}

func TestInstallWritesUnits(t *testing.T) {
	directory := t.TempDir()
	unitDirectory := makeInstallTestUnitDirectory(t, directory)
	runInstall(t, directory, unitDirectory)
	for _, name := range unitFileNames {
		assertSameAsAsset(t, filepath.Join(unitDirectory, name), name)
	}
}

func TestInstallKeepsExistingUnits(t *testing.T) {
	directory := t.TempDir()
	unitDirectory := makeInstallTestUnitDirectory(t, directory)
	existing := filepath.Join(unitDirectory, assets.DaemonUnitFileName)
	if err := os.WriteFile(existing, []byte(installTestExisting), unitFilePermission); err != nil {
		t.Fatal(err)
	}
	runInstall(t, directory, unitDirectory)
	if data, err := os.ReadFile(existing); err != nil || string(data) != installTestExisting {
		t.Fatalf(installTestKeptUnitFormat, data)
	}
	for _, name := range unitFileNames {
		if name == assets.DaemonUnitFileName {
			continue
		}
		assertSameAsAsset(t, filepath.Join(unitDirectory, name), name)
	}
}

func makeInstallTestUnitDirectory(t *testing.T, directory string) string {
	t.Helper()
	unitDirectory := filepath.Join(directory, installTestUnitDirectory)
	if err := os.Mkdir(unitDirectory, configDirectoryPermission); err != nil {
		t.Fatal(err)
	}
	return unitDirectory
}

func runInstall(t *testing.T, directory, unitDirectory string) {
	t.Helper()
	command := newRootCommand()
	command.SetArgs([]string{flagPrefix + flagConfig, filepath.Join(directory, installTestConfigName),
		commandInstall, flagPrefix + flagUnitDirectory, unitDirectory})
	command.SetOut(&bytes.Buffer{})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
}

func assertSameAsAsset(t *testing.T, path, name string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want, err := assets.Files.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf(installTestContentFormat, path)
	}
}
