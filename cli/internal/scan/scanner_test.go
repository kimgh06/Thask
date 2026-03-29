package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanFixture(t *testing.T) {
	dir := t.TempDir()

	// go.mod
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/testproject\n\ngo 1.22\n"), 0644)

	// main package
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(`package main

import (
	"example.com/testproject/auth"
	"example.com/testproject/handler"
	"fmt"
)

func main() {
	fmt.Println(auth.Name, handler.Name)
}
`), 0644)

	// auth package
	os.MkdirAll(filepath.Join(dir, "auth"), 0755)
	os.WriteFile(filepath.Join(dir, "auth", "auth.go"), []byte(`package auth

import "example.com/testproject/service"

var Name = service.Name
`), 0644)

	// handler package
	os.MkdirAll(filepath.Join(dir, "handler"), 0755)
	os.WriteFile(filepath.Join(dir, "handler", "handler.go"), []byte(`package handler

import (
	"example.com/testproject/auth"
	"example.com/testproject/service"
)

var Name = auth.Name + service.Name
`), 0644)

	// service package
	os.MkdirAll(filepath.Join(dir, "service"), 0755)
	os.WriteFile(filepath.Join(dir, "service", "service.go"), []byte(`package service

import "example.com/testproject/model"

var Name = model.Name
`), 0644)

	// cycle: service imports auth (auth already imports service)
	os.WriteFile(filepath.Join(dir, "service", "cycle.go"), []byte(`package service

import "example.com/testproject/auth"

var AuthRef = auth.Name
`), 0644)

	// model package
	os.MkdirAll(filepath.Join(dir, "model"), 0755)
	os.WriteFile(filepath.Join(dir, "model", "model.go"), []byte(`package model

import "example.com/testproject/auth"

var Name = auth.Name
`), 0644)

	result, err := Run(ScanOptions{Path: dir, MaxFiles: 500})
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// Should find 5 packages (main=root, auth, handler, service, model)
	if len(result.Nodes) != 5 {
		t.Errorf("expected 5 nodes, got %d", len(result.Nodes))
		for _, n := range result.Nodes {
			t.Logf("  node: %s", n.Title)
		}
	}

	// Should find edges
	if len(result.Edges) < 6 {
		t.Errorf("expected at least 6 edges, got %d", len(result.Edges))
		for _, e := range result.Edges {
			t.Logf("  edge: %s -> %s", e.SourceTitle, e.TargetTitle)
		}
	}

	// Mode should be merge
	if result.Mode != "merge" {
		t.Errorf("expected mode=merge, got %s", result.Mode)
	}
}

func TestScanMaxFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n\ngo 1.22\n"), 0644)

	// Create 5 Go files
	for i := 0; i < 5; i++ {
		os.WriteFile(filepath.Join(dir, fmt.Sprintf("file%d.go", i)), []byte("package main\n"), 0644)
	}

	_, err := Run(ScanOptions{Path: dir, MaxFiles: 3})
	if err == nil || !strings.Contains(err.Error(), "exceeded maxFiles") {
		t.Errorf("expected maxFiles error, got: %v", err)
	}
}

func TestScanNoGoMod(t *testing.T) {
	dir := t.TempDir()
	_, err := Run(ScanOptions{Path: dir})
	if err == nil || !strings.Contains(err.Error(), "no go.mod found") {
		t.Errorf("expected 'no go.mod found' error, got: %v", err)
	}
}
