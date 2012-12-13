package main

import (
	"os"
	"path/filepath"
	"path"
	"time"
	"ponyo.epfl.ch/go/get/optimization/go/optimization/log"
)

var dispatcherRepo *DispatcherRepo

type DispatcherRepo struct {
	scanned      map[string]time.Time
	cache        map[string]string
	lastScan     time.Time
	scanThrottle uint
}

func init() {
	dispatcherRepo = &DispatcherRepo{
		lastScan:     time.Unix(0, 0),
		scanThrottle: 10,
		scanned:      make(map[string]time.Time),
		cache:        make(map[string]string),
	}

	dispatcherRepo.scan()
}

func (x *DispatcherRepo) scanDir(p string) {
	log.W("Scanning for dispatchers in: %s", p)

	if when, ok := x.scanned[p]; ok {
		info, err := os.Stat(p)

		if err != nil {
			return
		}

		if info.ModTime().Before(when) {
			return
		}

		x.scanned[p] = info.ModTime()
	} else {
		x.scanned[p] = time.Now()
	}

	filepath.Walk(p, func(child string, info os.FileInfo, err error) error {
		if child == p {
			return nil
		}

		if err != nil || info.IsDir() {
			return filepath.SkipDir
		}

		if (info.Mode().Perm() & 0001) == 0 {
			return nil
		}

		log.W("Found possible dispatcher: %s", child)

		x.cache[path.Base(child)] = child
		return nil
	})
}

func (x *DispatcherRepo) scan() {
	if uint(time.Now().Sub(x.lastScan)/time.Second) < x.scanThrottle {
		return
	}

	dispdir := filepath.Join(AppConfig.LibExecDir, "liboptimization-dispatchers-2.0")
	x.scanDir(dispdir)

	epath := os.Getenv("OPTIMIZATION_DISPATCHERS_PATH")
	paths := filepath.SplitList(epath)

	for _, dir := range paths {
		x.scanDir(dir)
	}

	x.lastScan = time.Now()
}

func (x *DispatcherRepo) Find(name string) string {
	if filepath.IsAbs(name) {
		return name
	}

	x.scan()
	return x.cache[name]
}

func FindDispatcher(name string) string {
	return dispatcherRepo.Find(name)
}
