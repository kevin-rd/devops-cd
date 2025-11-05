package service

import (
	"encoding/json"
	"sort"
)

func decodeDependencyIDs(raw []byte) ([]int64, error) {
	if len(raw) == 0 {
		return []int64{}, nil
	}

	var ids []int64
	if err := json.Unmarshal(raw, &ids); err != nil {
		return nil, err
	}

	return ids, nil
}

func normalizeDependencyIDs(ids []int64) []int64 {
	if len(ids) == 0 {
		return []int64{}
	}

	seen := make(map[int64]struct{}, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}

	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func hasDependencyCycle(graph map[int64][]int64) bool {
	visited := make(map[int64]bool, len(graph))
	stack := make(map[int64]bool, len(graph))

	var dfs func(node int64) bool
	dfs = func(node int64) bool {
		if stack[node] {
			return true
		}
		if visited[node] {
			return false
		}

		visited[node] = true
		stack[node] = true

		for _, next := range graph[node] {
			if dfs(next) {
				return true
			}
		}

		stack[node] = false
		return false
	}

	for node := range graph {
		if dfs(node) {
			return true
		}
	}

	return false
}
