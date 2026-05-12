package remote

import (
	"strconv"
	"strings"
)

// ParseMeminfoAwkOutput parses the first line of awk output from /proc/meminfo
// shaped as "MemAvailable_kB SwapFree_kB" (two integers in kB). Returns ok=false
// when the line cannot be parsed.
func ParseMeminfoAwkOutput(line string) (memKB, swapKB int64, ok bool) {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) < 2 {
		return 0, 0, false
	}
	mkb, e1 := strconv.ParseInt(fields[0], 10, 64)
	skb, e2 := strconv.ParseInt(fields[1], 10, 64)
	if e1 != nil || e2 != nil {
		return 0, 0, false
	}
	return mkb, skb, true
}
