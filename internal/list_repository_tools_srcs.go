//go:build ignore
// +build ignore

// Copyright 2014 The Bazel Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// list_go_repository_tools prints Bazel labels for source files that
// gazelle and fetch_repo depend on. go_repository_tools resolves these
// labels so that when a source file changes, the gazelle and fetch_repo
// binaries are rebuilt for go_repository.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	dir := flag.String("dir", "", "directory to run in")
	check := flag.String("check", "", ".bzl file to check (relative to gazelle root)")
	generate := flag.String("generate", "", ".bzl file to generate (relative to gazelle root)")
	flag.Parse()
	if *check == "" && *generate == "" {
		log.Fatal("neither -check nor -generate were set")
	}
	if *check != "" && *generate != "" {
		log.Fatal("both -check and -generate were set")
	}
	if *dir != "" {
		if err := os.Chdir(filepath.FromSlash(*dir)); err != nil {
			log.Fatal(err)
		}
	}
	if *check != "" {
		*check = filepath.FromSlash(*check)
	}
	if *generate != "" {
		*generate = filepath.FromSlash(*generate)
	}

	var labels []string
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the tests directory since there is nothing that should be imported from there.
		if strings.HasPrefix(path, "tests/") {
			return filepath.SkipDir
		}

		base := filepath.Base(path)
		if base == "bcr_tests" || base == "docs" || base == "vendor" || base == "third_party" || base == "testdata" {
			return filepath.SkipDir
		}
		if !info.IsDir() &&
			(strings.HasSuffix(base, ".go") && !strings.HasSuffix(base, "_test.go") ||
				base == "BUILD.bazel" || base == "BUILD") {
			label := filepath.ToSlash(path)
			if i := strings.LastIndexByte(label, '/'); i >= 0 {
				label = fmt.Sprintf(`Label("//%s:%s")`, label[:i], label[i+1:])
			} else {
				label = fmt.Sprintf(`Label("//%s")`, label)
			}
			labels = append(labels, label)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	buf := &bytes.Buffer{}
	fmt.Fprintln(buf, "# Code generated by list_repository_tools_srcs.go; DO NOT EDIT.")
	fmt.Fprintln(buf, "# regenerate with `go run internal/list_repository_tools_srcs.go -dir $PWD -generate internal/go_repository_tools_srcs.bzl`")
	fmt.Fprintln(buf, "GO_REPOSITORY_TOOLS_SRCS = [")
	for _, label := range labels {
		fmt.Fprintf(buf, "    %s,\n", label)
	}
	fmt.Fprintln(buf, "]")

	if *generate != "" {
		if err := ioutil.WriteFile(*generate, buf.Bytes(), 0o666); err != nil {
			log.Fatal(err)
		}
	} else {
		got, err := ioutil.ReadFile(*check)
		if err != nil {
			log.Fatal(err)
		}
		if !bytes.Equal(got, buf.Bytes()) {
			log.Fatalf("generated file %s is not up to date", *check)
		}
	}
}
