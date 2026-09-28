//go:build !windows && !linux

package app

import (
	"errors"
	"os"
)

func lockFile(f *os.File) error {
	return errors.New("bloqueio de instância não suportado nesta plataforma")
}

func unlockFile(f *os.File) error { return nil }
