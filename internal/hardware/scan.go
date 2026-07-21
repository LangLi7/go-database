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
	OS        string  `json:"os"`
	Arch      string  `json:"arch"`
	CPU       CPUInfo `json:"cpu"`
	RAM       RAMInfo `json:"ram"`
	GPUs      []GPU   `json:"gpus"`
	Submitted bool    `json:"submitted,omitempty"` // true if sent remotely
}

// CPUInfo detailed processor data.
type CPUInfo struct {
	Model   string `json:"model"`
	Cores   int    `json:"cores"`   // physical cores
	Threads int    `json:"threads"` // logical (hyperthreading)
}

// RAMInfo memory details.
type RAMInfo struct {
	TotalGB int    `json:"total_gb"`
	Type    string `json:"type,omitempty"`    // DDR4/DDR5 (best-effort)
	SpeedMHz int   `json:"speed_mhz,omitempty"` // MT/s (best-effort)
}

// GPU describes a graphics card (VRAM is what matters for local models).
type GPU struct {
	Name        string `json:"name"`
	VRAMGB      int    `json:"vram_gb"`
	Driver      string `json:"driver,omitempty"`
}

// Scan collects host hardware info. CPU/RAM detail via OS tools
// (powershell on Windows, /proc + dmidecode on Linux).
// ponytail: best-effort fields (RAM Type/Speed, Driver) empty if unknown.
func Scan(_ context.Context) Spec {
	s := Spec{OS: runtime.GOOS, Arch: runtime.GOARCH}
	s.CPU = detectCPU()
	s.RAM = detectRAM()
	s.GPUs = detectGPUs()
	return s
}

func detectCPU() CPUInfo {
	ci := CPUInfo{Cores: runtime.NumCPU(), Threads: runtime.NumCPU()}
	switch runtime.GOOS {
	case "windows":
		if out, err := exec.Command("powershell", "-NoProfile", "-Command",
			"Get-CimInstance Win32_Processor | Select-Object Name,NumberOfCores,NumberOfLogicalProcessors | ConvertTo-Json").Output(); err == nil {
			ci.Model = psField(string(out), "Name")
			if v, e := strconv.Atoi(psField(string(out), "NumberOfCores")); e == nil {
				ci.Cores = v
			}
			if v, e := strconv.Atoi(psField(string(out), "NumberOfLogicalProcessors")); e == nil {
				ci.Threads = v
			}
		}
	case "linux":
		if b, err := os.ReadFile("/proc/cpuinfo"); err == nil {
			seen := false
			for _, line := range strings.Split(string(b), "\n") {
				if strings.HasPrefix(line, "model name") {
					ci.Model = strings.TrimSpace(strings.TrimPrefix(strings.SplitN(line, ":", 2)[1], " "))
					break
				}
				if strings.HasPrefix(line, "cpu cores") && !seen {
					if v, e := strconv.Atoi(strings.TrimSpace(strings.SplitN(line, ":", 2)[1])); e == nil {
						ci.Cores = v
						seen = true
					}
				}
			}
		}
	}
	return ci
}

func detectRAM() RAMInfo {
	ri := RAMInfo{TotalGB: 0}
	switch runtime.GOOS {
	case "windows":
		if out, err := exec.Command("powershell", "-NoProfile", "-Command",
			"Get-CimInstance Win32_PhysicalMemory | Select-Object Capacity,Speed,SMBIOSMemoryType | ConvertTo-Json").Output(); err == nil {
			ri.SpeedMHz = psInt(string(out), "Speed")
			ri.Type = ramType(psInt(string(out), "SMBIOSMemoryType"))
			// sum capacities (JSON may be array or single object)
			ri.TotalGB = psMemTotal(string(out))
		}
		// fallback: visible memory
		if ri.TotalGB == 0 {
			if out, err := exec.Command("wmic", "OS", "get", "TotalVisibleMemorySize", "/Value").Output(); err == nil {
				for _, line := range strings.Split(string(out), "\n") {
					if strings.HasPrefix(line, "TotalVisibleMemorySize=") {
						if kb, e := strconv.ParseInt(strings.TrimSpace(strings.TrimPrefix(line, "TotalVisibleMemorySize=")), 10, 64); e == nil {
							ri.TotalGB = int(kb / (1024 * 1024))
						}
					}
				}
			}
		}
	case "linux":
		if b, err := os.ReadFile("/proc/meminfo"); err == nil {
			for _, line := range strings.Split(string(b), "\n") {
				if strings.HasPrefix(line, "MemTotal:") {
					f := strings.Fields(line)
					if len(f) >= 2 {
						if kb, e := strconv.ParseInt(f[1], 10, 64); e == nil {
							ri.TotalGB = int(kb / (1024 * 1024))
						}
					}
				}
			}
		}
		if out, err := exec.Command("sudo", "dmidecode", "-t", "memory").Output(); err == nil {
			ri.Type = dmiRAMType(string(out))
			ri.SpeedMHz = dmiRAMSpeed(string(out))
		}
	}
	return ri
}

func detectGPUs() []GPU {
	var gpus []GPU
	if out, err := exec.Command("nvidia-smi", "--query-gpu=name,memory.total,driver_version",
		"--format=csv,noheader,nounits").Output(); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			f := strings.Split(line, ",")
			if len(f) >= 2 {
				g := GPU{Name: strings.TrimSpace(f[0])}
				if mb, e := strconv.ParseFloat(strings.TrimSpace(f[1]), 64); e == nil {
					g.VRAMGB = int(mb / 1024)
				}
				if len(f) >= 3 {
					g.Driver = strings.TrimSpace(f[2])
				}
				gpus = append(gpus, g)
			}
		}
	}
	return gpus
}

// ---- helpers ----

func psField(jsonStr, key string) string {
	// crude JSON field extraction (avoids encoding/json for tiny payloads)
	marker := "\"" + key + "\":"
	i := strings.Index(jsonStr, marker)
	if i < 0 {
		return ""
	}
	rest := jsonStr[i+len(marker):]
	rest = strings.TrimLeft(rest, " ")
	if strings.HasPrefix(rest, "\"") {
		end := strings.Index(rest[1:], "\"")
		if end >= 0 {
			return rest[1 : end+1]
		}
	}
	return ""
}

func psInt(jsonStr, key string) int {
	v, _ := strconv.Atoi(psField(jsonStr, key))
	return v
}

func psMemTotal(jsonStr string) int {
	// sum all "Capacity": "<bytes>" entries
	total := 0
	idx := 0
	for {
		marker := "\"Capacity\":"
		i := strings.Index(jsonStr[idx:], marker)
		if i < 0 {
			break
		}
		rest := jsonStr[idx+i+len(marker):]
		rest = strings.TrimLeft(rest, " ")
		end := strings.IndexAny(rest, ",}\n")
		if end < 0 {
			end = len(rest)
		}
		if b, e := strconv.ParseInt(strings.TrimSpace(rest[:end]), 10, 64); e == nil {
			total += int(b / (1024 * 1024 * 1024))
		}
		idx += i + len(marker)
	}
	return total
}

func ramType(smbios int) string {
	// SMBIOS memory type codes (subset)
	switch smbios {
	case 26:
		return "DDR4"
	case 34:
		return "DDR5"
	case 24:
		return "DDR3"
	case 35:
		return "DDR5" // LPDDR5
	case 33:
		return "DDR4" // LPDDR4
	default:
		return ""
	}
}

func dmiRAMType(out string) string {
	if strings.Contains(out, "DDR5") {
		return "DDR5"
	}
	if strings.Contains(out, "DDR4") {
		return "DDR4"
	}
	return ""
}

func dmiRAMSpeed(out string) int {
	// first "Speed: NNNN MT/s" line
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "Speed:") && strings.Contains(line, "MT/s") {
			f := strings.Fields(line)
			for i, w := range f {
				if strings.HasSuffix(w, "MT/s") {
					v, e := strconv.Atoi(strings.TrimSuffix(w, "MT/s"))
					if e == nil {
						return v
					}
				}
				_ = i
			}
		}
	}
	return 0
}
