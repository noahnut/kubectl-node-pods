package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/template"
)

type platform struct {
	OS     string `json:"os"`
	Arch   string `json:"arch"`
	Bin    string `json:"bin"`
	URI    string `json:"uri"`
	SHA256 string `json:"sha256"`
}

type manifestData struct {
	PluginName string
	Version    string
	Repo       string
	Platforms  []platform
}

func main() {
	var (
		templatePath = flag.String("template", "templates/krew-plugin.yaml.tmpl", "path to manifest template")
		outputPath   = flag.String("output", "node-pods.yaml", "output manifest path")
		pluginName   = flag.String("plugin-name", "", "krew plugin name")
		version      = flag.String("version", "", "plugin version (e.g. v0.1.1)")
		repo         = flag.String("repo", "", "github repo in owner/name format")
		platformsRaw = flag.String("platforms-json", "", "platform list in JSON")
	)
	flag.Parse()

	if *pluginName == "" || *version == "" || *repo == "" || *platformsRaw == "" {
		fatal("missing required flags: --plugin-name --version --repo --platforms-json")
	}

	var platforms []platform
	if err := json.Unmarshal([]byte(*platformsRaw), &platforms); err != nil {
		fatal(fmt.Sprintf("parse --platforms-json: %v", err))
	}
	if len(platforms) == 0 {
		fatal("platform list is empty")
	}

	for i, p := range platforms {
		if p.OS == "" || p.Arch == "" || p.Bin == "" || p.URI == "" || p.SHA256 == "" {
			fatal(fmt.Sprintf("invalid platform at index %d: all fields are required", i))
		}
	}

	sort.Slice(platforms, func(i, j int) bool {
		if platforms[i].OS == platforms[j].OS {
			return platforms[i].Arch < platforms[j].Arch
		}
		return platforms[i].OS < platforms[j].OS
	})

	tpl, err := template.ParseFiles(*templatePath)
	if err != nil {
		fatal(fmt.Sprintf("parse template: %v", err))
	}

	if err := os.MkdirAll(filepath.Dir(*outputPath), 0o755); err != nil {
		fatal(fmt.Sprintf("create output dir: %v", err))
	}
	f, err := os.Create(*outputPath)
	if err != nil {
		fatal(fmt.Sprintf("create output file: %v", err))
	}
	defer f.Close()

	data := manifestData{
		PluginName: *pluginName,
		Version:    *version,
		Repo:       *repo,
		Platforms:  platforms,
	}
	if err := tpl.Execute(f, data); err != nil {
		fatal(fmt.Sprintf("render template: %v", err))
	}
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
