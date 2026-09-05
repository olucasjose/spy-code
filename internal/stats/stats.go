// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package stats

import (
	"io/fs"
	"path/filepath"
	"strings"

	"tae/internal/filter"
)

type Data struct {
	Binary    map[string]int
	NonBinary map[string]int
}

func Calculate(baseRoot string, ignoreDirs []string) *Data {
	ignoreMap := make(map[string]bool)
	for _, dir := range ignoreDirs {
		ignoreMap[dir] = true
	}
	data := &Data{
		Binary:    make(map[string]int),
		NonBinary: make(map[string]int),
	}

	_ = filepath.WalkDir(baseRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			if ignoreMap[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		isBin, err := filter.IsBinaryFile(path)
		if err != nil {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext == "" {
			ext = "sem extensão"
		}

		if isBin {
			data.Binary[ext]++
		} else {
			data.NonBinary[ext]++
		}

		return nil
	})

	return data
}
