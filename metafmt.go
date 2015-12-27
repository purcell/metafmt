//
// Copyright (c) 2015 Lorenzo Villani
//
// Permission is hereby granted, free of charge, to any person obtaining a
// copy of this software and associated documentation files (the "Software"),
// to deal in the Software without restriction, including without limitation
// the rights to use, copy, modify, merge, publish, distribute, sublicense,
// and/or sell copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.
//

package main

import (
	"bytes"
	"flag"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ungerik/go-dry"
)

//
// Formatters
//

type formatter struct {
	Commands        [][]string
	EmacsMajorModes []string
	Extensions      []string
}

var formatters = []*formatter{
	// C/C++
	{
		Commands: [][]string{
			[]string{"clang-format", "-style=WebKit", "-"},
		},
		EmacsMajorModes: []string{"c-mode", "c++-mode"},
		Extensions:      []string{".c", ".cpp", ".cxx", ".h", ".hpp", ".hxx"},
	},
	// CSS
	{
		Commands: [][]string{
			[]string{"cssbeautify-bin", "--autosemicolon", "-f", "-"},
		},
		EmacsMajorModes: []string{"css-mode"},
		Extensions:      []string{".css"},
	},
	// Go
	{
		Commands: [][]string{
			[]string{"goimports"},
		},
		EmacsMajorModes: []string{"go-mode"},
		Extensions:      []string{".go"},
	},
	// JavaScript
	{
		Commands: [][]string{
			[]string{"semistandard-format", "-"},
		},
		EmacsMajorModes: []string{"js-mode", "js2-mode", "js3-mode"},
		Extensions:      []string{".js", ".jsx"},
	},
	// JSON
	{
		Commands: [][]string{
			[]string{"jsonlint", "-"},
		},
		EmacsMajorModes: []string{"json-mode"},
		Extensions:      []string{".json"},
	},
	// Python
	{
		Commands: [][]string{
			[]string{"autopep8", "--max-line-length=98", "-"},
			[]string{"isort", "--line-width", "98", "--multi_line", "3", "-"},
		},
		EmacsMajorModes: []string{"python-mode"},
		Extensions:      []string{".py"},
	},
	// SASS
	{
		Commands: [][]string{
			[]string{"sass-convert", "--no-cache", "--from", "sass", "--to", "sass", "--indent", "4", "--stdin"},
		},
		EmacsMajorModes: []string{"sass-mode"},
		Extensions:      []string{".sass"},
	},
	// SCSS
	{
		Commands: [][]string{
			[]string{"sass-convert", "--no-cache", "--from", "scss", "--to", "scss", "--indent", "4", "--stdin"},
		},
		EmacsMajorModes: []string{"scss-mode"},
		Extensions:      []string{".scss"},
	},
}

//
// Lookup maps
//

type lookupMap map[string]*formatter

var emacsToFormatter = make(lookupMap)
var extToFormatter = make(lookupMap)

func init() {
	for _, formatter := range formatters {
		for _, ext := range formatter.Extensions {
			extToFormatter[ext] = formatter
		}

		for _, majorMode := range formatter.EmacsMajorModes {
			emacsToFormatter[majorMode] = formatter
		}
	}
}

func formatterForEmacs() *formatter {
	if *emacs == "" {
		return nil
	}

	formatter, ok := emacsToFormatter[*emacs]
	if !ok {
		return nil
	}

	return formatter
}

func formatterForPath(path string) *formatter {
	ext := filepath.Ext(path)
	if ext == "" {
		return nil
	}

	fmt, ok := extToFormatter[ext]
	if !ok {
		return nil
	}

	return fmt
}

//
// Flags
//

var emacs = flag.String("emacs", "", "Emacs major mode")
var write = flag.Bool("write", false, "Write the file in place")

//
// Entry point
//

type formatOp func(string, *formatter) error

func main() {
	// Flags
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		return
	}

	// Format standard input, then stop
	if len(args) == 1 && args[0] == "-" {
		formatStdin()
		return
	}

	// Select mode of operation (format to file or standard output)
	var op formatOp
	if *write {
		op = formatWrite
	} else {
		op = formatStdout
	}

	// Format files
	for _, path := range args {
		if dry.FileIsDir(path) {
			formatDir(path, op)
		} else {
			formatFile(path, op)
		}
	}
}

//
// High level operations
//

var IgnoreDirs = []string{".git", ".hg", ".svn", "node_modules"}

func formatDir(path string, op formatOp) {
	filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() && dry.StringListContains(IgnoreDirs, info.Name()) {
			return filepath.SkipDir
		}

		if !info.IsDir() {
			formatFile(path, op)
		}

		return nil
	})
}

func formatFile(path string, op formatOp) {
	formatter := formatterForPath(path)
	if formatter == nil {
		return
	}

	if err := op(path, formatter); err != nil {
		log.Fatalln(err)
	}
}

func formatStdin() {
	formatter := formatterForEmacs()
	if formatter == nil {
		log.Fatalln("Must be given an Emacs major mode")
	}

	if err := formatChain(os.Stdout, os.Stdin, formatter.Commands); err != nil {
		log.Fatalln(err)
	}
}

//
// Low level operations
//

func formatWrite(path string, formatter *formatter) error {
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer file.Close()

	var buf bytes.Buffer

	if err := formatChain(&buf, file, formatter.Commands); err != nil {
		return err
	}

	if err := file.Truncate(0); err != nil {
		return err
	}

	if _, err := file.Seek(0, os.SEEK_SET); err != nil {
		return err
	}

	_, err = io.Copy(file, &buf)
	return err
}

func formatStdout(path string, formatter *formatter) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return formatChain(os.Stdout, file, formatter.Commands)
}

func formatChain(dst io.Writer, src io.Reader, commandChain [][]string) error {
	var buf, tmp bytes.Buffer

	for i, command := range commandChain {
		var stepSrc io.Reader

		if i == 0 {
			stepSrc = src
		} else {
			tmp.Reset()

			if _, err := io.Copy(&tmp, &buf); err != nil {
				return err
			}

			buf.Reset()

			stepSrc = &tmp
		}

		if err := format(&buf, stepSrc, command); err != nil {
			return err
		}
	}

	_, err := io.Copy(dst, &buf)
	return err
}

func format(dst io.Writer, src io.Reader, command []string) error {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdin = src
	cmd.Stdout = dst

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
