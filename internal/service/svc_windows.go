package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

// Install ensures a windows service is active
func Install(name, desc string) error {
	exepath, err := getExecPath(os.Args[0])
	if err != nil {
		return err
	}

	return install(name, desc, exepath)
}

func getExecPath(exepath string) (string, error) {
	p, err := filepath.Abs(exepath)
	if err != nil {
		return "", err
	}

	fi, err := os.Stat(p)
	if err == nil {
		if fi.Mode().IsRegular() {
			return p, nil
		}
	}

	if 0 == len(filepath.Ext(p)) {
		var err error
		p += ".exe"
		fi, err = os.Stat(p)
		if nil != err {
			return "", err
		}
	}

	if !fi.Mode().IsRegular() {
		// this should never happen
		return "", errors.New("not a regular file")
	}

	return p, nil
}

func install(name, desc, exepath string) error {
	m, err := mgr.Connect()
	if nil != err {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if nil == err {
		s.Close()
		return nil
	}

	s, err = m.CreateService(name, exepath, mgr.Config{DisplayName: desc}, "is", "auto-started")
	if err != nil {
		return err
	}
	defer s.Close()

	err = eventlog.InstallAsEventCreate(name, eventlog.Error|eventlog.Warning|eventlog.Info)
	if nil != err {
		s.Delete()
		return fmt.Errorf("could not install system service: %v", err)
	}

	return nil
}
