package main

import (
	"os"
	"path/filepath"
	"time"
)

var dispatcherRepo *DispatcherRepo

type DispatcherRepo struct {
	scanned      map[string]time.Time
	cache        map[string]string
	lastScan     time.Time
	scanThrottle uint
}

func init() {
	dispatcherRepo := &DispatcherRepo{
		lastScan:     time.Unix(0, 0),
		scanThrottle: 10,
		scanned:      make(map[string]time.Time),
		cache:        make(map[string]string),
	}

	dispatcherRepo.scan()
}

func (x *DispatcherRepo) scanDir(path string) {
	if when, ok := x.scanned[path]; ok {
		info, err := os.Stat(path)

		if err != nil {
			return
		}

		if info.ModTime().Before(when) {
			return
		}

		x.scanned[path] = info.ModTime()
	} else {
		filepath.Walk(path, func(child string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return filepath.SkipDir
			}

			x.cache[child] = filepath.Join(path, child)
			return nil
		})

		x.scanned[path] = time.Now()
	}
}

func (x *DispatcherRepo) scan() {
	if uint(time.Now().Sub(x.lastScan)/time.Second) < x.scanThrottle {
		return
	}

	dispdir := filepath.Join(AppConfig.Libexecdir, "liboptimization-dispatchers-2.0")
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
