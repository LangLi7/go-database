package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Println(`go-database Launcher

Nutze:
  go run ./cmd/godb           → Server bauen + starten
  go run ./cmd/godb mcp       → MCP-Server bauen + starten
  go run ./cmd/godb build     → Nur bauen (Server + MCP)

Umgebungsvariablen: GODB_MCP_ENABLED, GODB_MCP_API_KEY, GODB_LOG_LEVEL usw.`)
		return
	}

	target := "./cmd/server"
	out := "bin/godb-server.exe"
	if len(args) > 0 {
		switch args[0] {
		case "mcp":
			target = "./cmd/mcp"
			out = "bin/godb-mcp.exe"
		case "build":
			build("./cmd/server", "bin/godb-server.exe")
			build("./cmd/mcp", "bin/godb-mcp.exe")
			fmt.Println("✓ Build abgeschlossen")
			return
		}
	}

	fmt.Printf("🔨 Baue %s → %s\n", target, out)
	build(target, out)

	fmt.Printf("🚀 Starte %s\n", target)
	run := exec.Command(out)
	run.Stdout = os.Stdout
	run.Stderr = os.Stderr
	run.Stdin = os.Stdin
	if err := run.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Fehler: %v\n", err)
		os.Exit(1)
	}
}

func build(target, out string) {
	cmd := exec.Command("go", "build", "-o", out, target)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Build fehlgeschlagen: %v\n", err)
		os.Exit(1)
	}
}
