package main

import (
	"sort"
	"strings"
)

func factsToStr(userData map[string]string) string {
	if len(userData) == 0 {
		return "\n\n"
	}
	keys := make([]string, 0, len(userData))
	for k := range userData {
		if k == "state" || k == "choice" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+" - "+userData[k])
	}
	return "\n" + strings.Join(parts, "\n") + "\n"
}
