package hardware

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// Spec describes the host hardware relevant for local LLM/model selection.
type Spec struct {
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	CPU       int    `json:"cpu_cores"`
	RAMGB     int    `json:"ram_gb"`
	GPUs      []GPU  `json:"gpus"`
}

// GPU describes a graphics card (VRAM is what matters for local models).
type GPU struct {
	Name   string `json:"name"`
	VRAMGB int    `json:"vram_gb"`
}

// Scan collects host hardware info. RAM is read from cgroup/proc on Linux and
// from `sysctl` on macOS; Windows falls back to runtime total (best-effort).
// ponytail: VRAM via `nvidia-smi` when present, else empty (CPU-only inference).
func Scan(_ context.Context) Spec {
	s := Spec{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
		CPU:  runtime.NumCPU(),
	}
	s.RAMGB = detectRAM()
	s.GPUs = detectGPUs()
	return s
}

func detectRAM() int {
	// Linux: MemTotal from /proc/meminfo (kB)
	if b, err := os.ReadFile("/proc/meminfo"); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				f := strings.Fields(line)
				if len(f) >= 2 {
					kb, _ := strconv.ParseInt(f[1], 10, 64)
					return int(kb / (1024 * 1024))
				}
			}
		}
	}
	// macOS
	if out, err := exec.Command("sysctl", "-n", "hw.memsize").Output(); err == nil {
		if b, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64); err == nil {
			return int(b / (1024 * 1024 * 1024))
		}
	}
	// Windows: wmic
	if out, err := exec.Command("wmic", "OS", "get", "TotalVisibleMemorySize", "/Value").Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(line, "TotalVisibleMemorySize=") {
				kb, _ := strconv.ParseInt(strings.TrimSpace(strings.TrimPrefix(line, "TotalVisibleMemorySize=")), 10, 64)
				return int(kb / (1024 * 1024))
			}
		}
	}
	return 0
}

func detectGPUs() []GPU {
	var gpus []GPU
	// nvidia-smi (Linux/Windows)
	if out, err := exec.Command("nvidia-smi", "--query-gpu=name,memory.total", "--format=csv,noheader,nounits").Output(); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			f := strings.Split(line, ",")
			if len(f) == 2 {
				mb, _ := strconv.ParseFloat(strings.TrimSpace(f[1]), 64)
				gpus = append(gpus, GPU{Name: strings.TrimSpace(f[0]), VRAMGB: int(mb / 1024)})
			}
		}
	}
	return gpus
}

// Scan host hardware (see Scan for details).
