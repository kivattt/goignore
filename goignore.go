package goignore

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func bufferLengthForPathComponents() int {
	return 2048
}

func maxPathLength() int {
	// You need atleast 1 character between each separator character for mySplitBuf() to use up a path component
	// e.g. "a/a/a/a/a/..."
	return 2 * bufferLengthForPathComponents()
}

// this is my own implementation of strings.Split()
// for my use case, this is way faster than the stdlib one
// the function expects a slice of sufficient length to get passed to it,
// this avoids unnecessary memory allocation
func mySplitBuf(s string, sep byte, pathComponentsBuf []string) []string {
	idx := 0
	l := 0
	for {
		pos := strings.IndexByte(s[l:], sep)

		if pos == -1 {
			break
		}

		absolutePos := l + pos
		if absolutePos > l {
			pathComponentsBuf[idx] = s[l:absolutePos]
			idx++
		}
		l = absolutePos + 1
	}
	// handle the last part separately
	if l < len(s) {
		pathComponentsBuf[idx] = s[l:]
		idx++
	}

	// truncate the slice to the actual number of components
	return pathComponentsBuf[:idx]
}

// this is my own implementation of strings.Split()
// for my use case, this is better than the stdlib one
func mySplit(s string, sep byte) []string {
	l := 0
	buf := make([]string, 0, 32)
	for {
		pos := strings.IndexByte(s[l:], sep)

		if pos == -1 {
			break
		}

		absolutePos := l + pos
		if absolutePos > l {
			buf = append(buf, s[l:absolutePos])
		}
		l = absolutePos + 1
	}

	// handle the last part separately
	if l < len(s) {
		buf = append(buf, s[l:])
	}

	return buf
}

type ruleInstructionType byte

const (
	raw ruleInstructionType = iota
	star
	starStar
	questionmark
	charClass
)

type ruleInstruction struct {
	Type ruleInstructionType
	Data []byte
}

type ruleComponent struct {
	Instructions []ruleInstruction
	Starstar     bool
	Star         bool
}

// Represents a single rule in a .gitignore file
// Components is a list of path components to match against
// Negate is true if the rule negates the match (i.e. starts with '!')
// OnlyDirectory is true if the rule matches only directories (i.e. ends with '/')
// Relative is true if the rule is relative (i.e. starts with '/')
type rule struct {
	Components    []ruleComponent
	Negate        bool
	OnlyDirectory bool
	Relative      bool
}

func selectorMatch(c byte, selector string) bool {
	switch selector {
	case "alnum":
		return ('0' <= c && c <= '9') || ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z')
	case "alpha":
		return ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z')
	case "blank":
		return c == ' ' || c == '\t'
	case "cntrl":
		return c < 32 || c == 127
	case "digit":
		return '0' <= c && c <= '9'
	case "graph":
		return 33 <= c && c <= 126
	case "lower":
		return 'a' <= c && c <= 'z'
	case "print":
		return 32 <= c && c <= 126
	case "punct":
		return (33 <= c && c <= 47) || (58 <= c && c <= 64) || (91 <= c && c <= 96) || (123 <= c && c <= 126)
	case "space":
		return (9 <= c && c <= 13) || c == 32
	case "upper":
		return 'A' <= c && c <= 'Z'
	case "xdigit":
		return ('0' <= c && c <= '9') || ('A' <= c && c <= 'F') || ('a' <= c && c <= 'f')
	default:
		return false
	}
}

func makeRuleComponent(component string) (ruleComponent, error) {
	instructions := make([]ruleInstruction, 0, 8)
	r := 0

	if component == "*" {
		instructions = append(instructions, ruleInstruction{
			Type: star,
		})
		return ruleComponent{
			Instructions: instructions,
			Starstar:     false,
			Star:         true,
		}, nil
	}
	if component == "**" {
		instructions = append(instructions, ruleInstruction{
			Type: starStar,
		})
		return ruleComponent{
			Instructions: instructions,
			Starstar:     true,
			Star:         false,
		}, nil
	}

	for r < len(component) {
		switch component[r] {
		case '*':
			r++
			instructions = append(instructions, ruleInstruction{
				Type: star,
			})
			continue
		case '?':
			r++
			instructions = append(instructions, ruleInstruction{
				Type: questionmark,
			})
			continue
		case '[':
			r++
			charC := ruleInstruction{
				Type: charClass,
				Data: make([]byte, 256), // maybe get the size of byte instead? It really doesn't matter
			}

			if r >= len(component) {
				return ruleComponent{}, errors.New("unclosed character class")
			}

			negate := false

			if component[r] == '!' || component[r] == '^' {
				negate = true
				r++
				if r >= len(component) {
					return ruleComponent{}, errors.New("unclosed character class")
				}
			}

			// special-case leading ']'
			if component[r] == ']' {
				charC.Data[']'] = 1
				r++
			}

			for r < len(component) && component[r] != ']' {
				// handle escaping
				if component[r] == '\\' && r+1 < len(component) {
					r += 2
					continue
				}
				// handle special [:class:] character classes
				if r+2 < len(component) && component[r] == '[' && component[r+1] == ':' {
					r += 2
					s := r
					for s < len(component) && (component[s] != ']' || component[s-1] != ':') {
						s++
					}

					if s >= len(component) || s < r+2 {
						return ruleComponent{}, errors.New("unclosed character class")
					}

					selector := component[r : s-1]

					for i := 0; i < 256; i++ {
						if selectorMatch(byte(i), selector) {
							charC.Data[i] = 1
						}
					}

					r = s + 1
					continue
				}
				// handle ranges
				if r+2 < len(component) && component[r+1] == '-' && component[r+2] != ']' {
					a := component[r]
					b := component[r+2]
					if a <= b {
						for i := a; i < b; i++ {
							charC.Data[i] = 1
						}
						charC.Data[b] = 1
					}
					r += 3
					continue
				}
				// add to LUT
				charC.Data[component[r]] = 1
				r++
			}

			if r >= len(component) || component[r] != ']' {
				return ruleComponent{}, errors.New("unclosed character class")
			}

			r++ // skip closing ']'

			if negate {
				for i := 0; i < len(charC.Data); i++ {
					charC.Data[i] = 1 - charC.Data[i]
				}
			}

			instructions = append(instructions, charC)
			continue
		}

		rawPattern := make([]byte, 0, 16)

		for r < len(component) && component[r] != '*' && component[r] != '?' && component[r] != '[' {
			if component[r] == '\\' && r+1 < len(component) {
				rawPattern = append(rawPattern, component[r+1])
				r += 2
				continue
			}
			rawPattern = append(rawPattern, component[r])
			r++
		}

		instructions = append(instructions, ruleInstruction{
			Type: raw,
			Data: rawPattern,
		})
	}

	return ruleComponent{
		Instructions: instructions,
		Starstar:     false,
		Star:         false,
	}, nil
}

func stringMatch(str string, component ruleComponent) bool {
	// i is the index in str, j is the index in pattern
	i, j := 0, 0
	lastStarIdx := -1
	lastStrIdx := -1

	for i < len(str) {
		if j < len(component.Instructions) {
			instruction := component.Instructions[j]
			switch instruction.Type {
			case questionmark:
				i++
				j++
				continue
			case star:
				lastStarIdx = j
				lastStrIdx = i
				j++
				continue
			case starStar:
				return true
			case charClass:
				if instruction.Data[str[i]] == 1 {
					i++
					j++
					continue
				}
			case raw:
				pattern := string(instruction.Data)
				if i+len(pattern) > len(str) {
					break
				}
				if str[i:i+len(pattern)] == pattern {
					i += len(pattern)
					j++
					continue
				}
			}
		}

		if lastStarIdx != -1 {
			j = lastStarIdx + 1
			lastStrIdx++
			i = lastStrIdx
			continue
		}

		// we can't backtrack, so no match
		return false
	}

	// consume remaining stars in component
	for j < len(component.Instructions) && component.Instructions[j].Type == star {
		j++
	}

	// if we ran out of instructions, return true
	return j >= len(component.Instructions)
}

// Tries to match the path components against the rule components
// matches is true if the path matches the rule, final is true if the rule matched the whole path
// the final parameter is used for rules that match directories only
func matchComponents(path []string, components []ruleComponent) (matches bool, final bool) {
	i := 0
	for ; i < len(components); i++ {
		if i >= len(path) {
			// we ran out of path components, but still have components to match
			return false, false
		}
		if components[i].Starstar {
			// stinky recursive step
			for j := len(path) - 1; j >= i; j-- {
				match, final := matchComponents(path[j:], components[i+1:])
				if match {
					// pass final trough
					return true, final
				}
			}
			return false, false
		}

		if !stringMatch(path[i], components[i]) {
			return false, false
		}
	}
	return true, i == len(path) // if we matched all components, check if we are at the end of the path
}

// Tries to match the path against the rule
// the function expects a buffer of sufficient size to get passed to it, this avoids excessive memory allocation
func (r *rule) matchesPath(isDirectory bool, pathComponents []string) bool {
	if !r.Relative {
		// stinky recursive step
		for j := 0; j < len(pathComponents); j++ {
			match, final := matchComponents(pathComponents[j:], r.Components)
			if match {
				return !r.OnlyDirectory || r.OnlyDirectory && (!final || final && isDirectory)
			}
		}

		return false
	}

	match, final := matchComponents(pathComponents, r.Components)

	return match && (!r.OnlyDirectory || r.OnlyDirectory && (!final || final && isDirectory))
}

// Stores a list of rules for matching paths against .gitignore patterns
// PathComponentsBuf is a temporary buffer for mySplit calls, this avoids excessive allocation
type GitIgnore struct {
	rules             []rule
	pathComponentsBuf []string
}

// Creates a Gitignore from a list of patterns (lines in a .gitignore file)
func CompileIgnoreLines(patterns []string) *GitIgnore {
	gitignore := &GitIgnore{
		rules:             make([]rule, 0, len(patterns)),
		pathComponentsBuf: make([]string, bufferLengthForPathComponents()),
	}

	for _, pattern := range patterns {
		// skip empty lines, comments, and trailing/leading whitespace
		pattern = strings.Trim(pattern, " \t\r\n")
		if pattern == "" || pattern == "!" || pattern[0] == '#' {
			continue
		}

		rule := createRule(pattern)

		gitignore.rules = append(gitignore.rules, rule)
	}

	return gitignore
}

// Same as CompileIgnoreLines, but reads from a file
func CompileIgnoreFile(filename string) (*GitIgnore, error) {
	lines, err := os.ReadFile(filename)

	if err != nil {
		return nil, err
	}
	return CompileIgnoreLines(strings.Split(string(lines), "\n")), nil
}

// create a rule from a pattern
func createRule(pattern string) rule {
	negate := false
	onlyDirectory := false
	relative := false
	if pattern[0] == '!' {
		negate = true
		pattern = pattern[1:] // skip the '!'
	}

	if pattern[0] == '/' {
		relative = true
		pattern = pattern[1:] // skip the '/'
	}

	// check if the pattern ends with a '/', which means it only matches directories
	if len(pattern) > 0 && pattern[len(pattern)-1] == '/' {
		onlyDirectory = true
	}

	// split the pattern into components
	// we use the default split function because this only runs once for each rule
	// this saves memory compared to using mySplit
	components := mySplit(pattern, '/')

	ruleComponents := make([]ruleComponent, len(components))

	for i := 0; i < len(components); i++ {
		comp, err := makeRuleComponent(components[i])
		if err == nil {
			ruleComponents[i] = comp
		}
	}

	return rule{
		Components:    ruleComponents,
		Negate:        negate,
		OnlyDirectory: onlyDirectory,
		Relative:      relative || len(components) > 1,
	}
}

func (g *GitIgnore) matchesPathNoError(path string) bool {
	result, _ := g.MatchesPath(path)
	return result
}

// Tries to match the path to all the rules in the gitignore
// Returns an error if the path is longer than 4096 bytes.
func (g *GitIgnore) MatchesPath(path string) (bool, error) {
	// Guard against out-of-bounds panic in mySplitBuf()
	if len(path) > maxPathLength() {
		return false, errors.New("path cannot be longer than " + strconv.Itoa(maxPathLength()) + " bytes")
	}

	// TODO: check if path actually points to a directory on the filesystem
	isDir := strings.HasSuffix(path, "/")
	path = filepath.Clean(path)
	path = filepath.ToSlash(path)
	if path == "." {
		path = "/"
		isDir = true
	}
	if !fs.ValidPath(path) {
		return false, nil
	}
	pathComponents := mySplit(path, '/')
	matched := false

	for _, rule := range g.rules {
		if rule.matchesPath(isDir, pathComponents) {
			if !rule.Negate {
				matched = true
			} else {
				matched = false
			}
		}
	}
	return matched, nil
}
