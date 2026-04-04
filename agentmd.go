package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
)

const sourceFile = "AGENTS.md"

var allFiles = []string{"AGENTS.md", "CLAUDE.md", "GEMINI.md"}
var symlinkFiles = []string{"CLAUDE.md", "GEMINI.md"}

type fileState int

const (
	missing        fileState = iota
	regularFile              // regular file (or dir, etc.)
	symlinkCorrect           // symlink pointing to AGENTS.md
	symlinkWrong             // symlink pointing elsewhere
)

func classify(name string) (fileState, error) {
	info, err := os.Lstat(name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return missing, nil
		}
		return missing, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(name)
		if err != nil {
			return symlinkWrong, nil
		}
		if target == sourceFile {
			return symlinkCorrect, nil
		}
		return symlinkWrong, nil
	}
	if !info.Mode().IsRegular() {
		return missing, fmt.Errorf("%s is not a regular file (is a directory or special file)", name)
	}
	return regularFile, nil
}

func audit() (map[string]fileState, error) {
	m := make(map[string]fileState, len(allFiles))
	for _, f := range allFiles {
		state, err := classify(f)
		if err != nil {
			return nil, err
		}
		m[f] = state
	}
	return m, nil
}

func run(check bool) error {
	states, err := audit()
	if err != nil {
		return err
	}

	// Collect regular files
	var regulars []string
	for _, f := range allFiles {
		if states[f] == regularFile {
			regulars = append(regulars, f)
		}
	}

	// Step 2: Determine source of truth
	switch {
	case len(regulars) == 0:
		// Case A: no files exist
		if check {
			fmt.Println("AGENTS.md does not exist")
			return fmt.Errorf("state is incorrect")
		}
		if err := os.WriteFile(sourceFile, []byte{}, 0644); err != nil {
			return err
		}
		fmt.Println("created AGENTS.md")

	case len(regulars) == 1 && regulars[0] == sourceFile:
		// Case B: only AGENTS.md exists as regular file — good

	case len(regulars) == 1 && regulars[0] != sourceFile:
		// Case C: one non-AGENTS.md regular file
		name := regulars[0]
		if check {
			fmt.Printf("%s exists but AGENTS.md does not\n", name)
			return fmt.Errorf("state is incorrect")
		}
		if err := os.Rename(name, sourceFile); err != nil {
			return err
		}
		fmt.Printf("renamed %s → AGENTS.md\n", name)

	default:
		// Multiple regular files — check content
		contents := make(map[string][]byte, len(regulars))
		for _, f := range regulars {
			data, err := os.ReadFile(f)
			if err != nil {
				return err
			}
			contents[f] = data
		}

		allSame := true
		first := contents[regulars[0]]
		for _, f := range regulars[1:] {
			if !bytes.Equal(first, contents[f]) {
				allSame = false
				break
			}
		}

		if allSame {
			// Case D: identical content
			if check {
				fmt.Printf("multiple regular files exist: %v\n", regulars)
				return fmt.Errorf("state is incorrect")
			}
			// Pick AGENTS.md if present, otherwise first alphabetically
			chosen := regulars[0]
			for _, f := range regulars {
				if f == sourceFile {
					chosen = f
					break
				}
			}
			if chosen != sourceFile {
				if err := os.Rename(chosen, sourceFile); err != nil {
					return err
				}
				fmt.Printf("renamed %s → AGENTS.md\n", chosen)
			}
		} else {
			// Case E: different content — conflict
			if check {
				fmt.Println("conflict: the following files have different content:")
				for _, f := range regulars {
					fmt.Printf("  - %s (%d bytes)\n", f, len(contents[f]))
				}
				return fmt.Errorf("state is incorrect")
			}
			if err := resolveConflict(regulars, contents); err != nil {
				return err
			}
		}
	}

	// Step 3: Ensure symlinks
	return ensureSymlinks(states, check)
}

func resolveConflict(regulars []string, contents map[string][]byte) error {
	sort.Strings(regulars)
	fmt.Println("conflict: the following files exist with different content:")
	for i, f := range regulars {
		fmt.Printf("  [%d] %s (%d bytes)\n", i+1, f, len(contents[f]))
	}
	fmt.Print("which file should be the source of truth? ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return fmt.Errorf("no input, aborting")
	}
	line := scanner.Text()
	idx, err := strconv.Atoi(line)
	if err != nil || idx < 1 || idx > len(regulars) {
		return fmt.Errorf("invalid choice: %s", line)
	}

	chosen := regulars[idx-1]

	// Remove all other regular files
	for _, f := range regulars {
		if f == chosen {
			continue
		}
		if err := os.Remove(f); err != nil {
			return err
		}
		fmt.Printf("removed %s\n", f)
	}

	// Rename chosen to AGENTS.md if needed
	if chosen != sourceFile {
		if err := os.Rename(chosen, sourceFile); err != nil {
			return err
		}
		fmt.Printf("renamed %s → AGENTS.md\n", chosen)
	}

	return nil
}

func ensureSymlinks(states map[string]fileState, check bool) error {
	// Re-audit since we may have changed things
	if !check {
		var err error
		states, err = audit()
		if err != nil {
			return err
		}
	}

	incorrect := false
	for _, f := range symlinkFiles {
		switch states[f] {
		case symlinkCorrect:
			continue
		case missing:
			if check {
				fmt.Printf("%s is missing\n", f)
				incorrect = true
				continue
			}
			if err := os.Symlink(sourceFile, f); err != nil {
				return err
			}
			fmt.Printf("%s → AGENTS.md\n", f)
		default:
			// wrong symlink or regular file
			if check {
				if states[f] == symlinkWrong {
					fmt.Printf("%s is a symlink to the wrong target\n", f)
				} else {
					fmt.Printf("%s is a regular file, not a symlink\n", f)
				}
				incorrect = true
				continue
			}
			if err := os.Remove(f); err != nil {
				return err
			}
			if err := os.Symlink(sourceFile, f); err != nil {
				return err
			}
			fmt.Printf("%s → AGENTS.md\n", f)
		}
	}

	if check && incorrect {
		return fmt.Errorf("state is incorrect")
	}
	return nil
}
