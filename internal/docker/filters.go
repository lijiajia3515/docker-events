package docker

import (
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/filters"
)

func buildFilterArgs(rawFilters, eventTypes []string) (filters.Args, error) {
	args := filters.NewArgs()

	for _, eventType := range eventTypes {
		t := strings.TrimSpace(eventType)
		if t == "" {
			continue
		}
		args.Add("type", strings.ToLower(t))
	}

	for _, rawFilter := range rawFilters {
		filter := strings.TrimSpace(rawFilter)
		if filter == "" {
			continue
		}
		parts := strings.SplitN(filter, "=", 2)
		if len(parts) != 2 {
			return filters.Args{}, fmt.Errorf("无效的 Docker 过滤器 %q，期望格式为 key=value", rawFilter)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" || value == "" {
			return filters.Args{}, fmt.Errorf("无效的 Docker 过滤器 %q，期望格式为 key=value", rawFilter)
		}
		args.Add(key, value)
	}

	return args, nil
}
