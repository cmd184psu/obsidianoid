package main

import (
	"flag"
	"fmt"
	"os"

	"obsidianoid/internal/config"
	"obsidianoid/internal/server"
)

func main() {
	cfgPath := flag.String("config", config.DefaultPath(), "path to config file")
	genCert := flag.Bool("gen-cert", false, "generate a self-signed TLS certificate and exit")
	insecure := flag.Bool("insecure", false, "run plain HTTP instead of HTTPS (not recommended)")
	flag.Parse()

	if *genCert {
		if err := generateSelfSignedCert(server.CertDir()); err != nil {
			fmt.Fprintf(os.Stderr, "❌ cert generation failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Self-signed cert written to", server.CertDir())
		return
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ config error: %v\n", err)
		fmt.Fprintf(os.Stderr, "   Create %s with at minimum: {\"vault_path\":\"/path/to/vault\"}\n", *cfgPath)
		os.Exit(1)
	}

	h := server.New(cfg)

	if *insecure {
		if err := server.RunInsecure(cfg, h); err != nil {
			fmt.Fprintf(os.Stderr, "❌ server error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := server.Run(cfg, h); err != nil {
		fmt.Fprintf(os.Stderr, "❌ server error: %v\n", err)
		os.Exit(1)
	}
}
