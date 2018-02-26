package msi

import "github.com/modulesio/butler/installer"

type Manager struct {
}

var _ installer.Manager = (*Manager)(nil)

func (m *Manager) Name() string {
	return "msi"
}

func Register() {
	installer.RegisterManager(&Manager{})
}
