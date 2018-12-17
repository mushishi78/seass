package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/BurntSushi/toml"
)

func main() {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	errors, err := Lint(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	for _, err := range errors {
		fmt.Fprintln(os.Stderr, err)
	}
}

type config struct {
	Ignore []string
}

// Lint takes a project folder looks for SeaSS errors
func Lint(dir string) ([]string, error) {
	errors := make(map[string]struct{})

	// Read seass.toml config file

	contents, err := ioutil.ReadFile(dir + "/seass.toml")
	if err != nil {
		return nil, fmt.Errorf("failed to read seass.toml file: %v", err)
	}

	var conf config
	if _, err := toml.Decode(string(contents), &conf); err != nil {
		return nil, fmt.Errorf("failed to parse seass.toml file: %v", err)
	}

	// Collect all the files in the directory

	files := make([]string, 0)

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relativePath := strings.TrimPrefix(path, dir+"/")

		for _, ignore := range conf.Ignore {
			if relativePath == ignore {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if strings.HasSuffix(relativePath, ".css") {
			files = append(files, relativePath)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %v", err)
	}

	possibleDuplicates := make(map[string]string)

	// Parse the css files

	for _, relativePath := range files {
		file, err := os.Open(dir + "/" + relativePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read css file: %v", err)
		}

		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanRunes)
		lineStart := 1
		columnStart := 1
		lineEnd := 1
		columnEnd := 0
		token := ""
		openQuote := false
		openComment := false
		openDefinition := false

		nextToken := func() {
			lineStart = lineEnd
			columnStart = columnEnd
			token = ""
		}

		for scanner.Scan() {
			ch := scanner.Text()
			token = token + ch

			columnEnd = columnEnd + 1
			if ch == "\n" {
				lineEnd = lineEnd + 1
				columnEnd = 1
			}

			// Quote
			if openQuote {
				if strings.HasSuffix(token, "\\\"") || strings.HasSuffix(token, "\\'") {
					continue
				}
				if ch == "\"" || ch == "'" {
					openQuote = false
				}
				continue
			}
			if !openComment && ch == "\"" || ch == "'" {
				openQuote = true
				continue
			}

			// Comments
			if openComment {
				if strings.HasSuffix(token, "*/") {
					i := strings.Index(token, "/*")
					token = token[:i]
					openComment = false
				}
				continue
			}
			if strings.HasSuffix(token, "/*") {
				openComment = true
				continue
			}

			// Ignore leading whitespace
			if len(token) == 1 && unicode.IsSpace(rune(ch[0])) {
				nextToken()
				continue
			}

			// Ignore closing block, incase inside a rule block like @media
			if token == "}" {
				nextToken()
				continue
			}

			// At statements
			if strings.HasPrefix(token, "@charset") ||
				strings.HasPrefix(token, "@import") ||
				strings.HasPrefix(token, "@namespace") {

				if ch == ";" {
					nextToken()
				}
				continue
			}

			// At rule blocks
			if strings.HasPrefix(token, "@document") ||
				strings.HasPrefix(token, "@media") {

				if ch == "{" {
					nextToken()
				}
				continue
			}

			// At definition blocks
			if strings.HasPrefix(token, "@font-face") ||
				strings.HasPrefix(token, "@page") {

				if ch == "}" {
					nextToken()
				}
				continue
			}

			// At keyframes blocks
			if strings.HasPrefix(token, "@keyframes") {
				openingBraces := 0
				closingBraces := 0

				for _, r := range token {
					if r == '{' {
						openingBraces++
					}
					if r == '}' {
						closingBraces++
					}
				}

				if openingBraces > 0 && openingBraces == closingBraces {
					nextToken()
				}

				continue
			}

			// Qualified rules
			if openDefinition {
				if strings.HasSuffix(token, "}") {
					openDefinition = false
					nextToken()
				}
				continue
			}
			if strings.HasSuffix(token, "{") {
				openDefinition = true
				prelude := strings.TrimSpace(token[:len(token)-1])

				err := func(message string) string {
					return fmt.Sprintf("%v:%v:%v-%v:%v - %v", relativePath, lineStart, columnStart, lineEnd, columnEnd-2, message)
				}

				// Parse and remove attribute selectors
				withoutBrackets := ""
				escaping := false
				openQuote := false
				openSquare := false
				openParens := false
				for _, r := range prelude {
					if openQuote {
						if r == '\\' {
							escaping = true
							continue
						}
						if !escaping && (r == '"' || r == '\'') {
							openQuote = false
						}
						escaping = false
						continue
					}
					if r == '"' || r == '\'' {
						openQuote = true
					}
					if openSquare {
						if r == ']' {
							openSquare = false
						}
						continue
					}
					if r == '[' {
						openSquare = true
						errors[err("attribute selector not allowed")] = struct{}{}
						continue
					}
					if openParens {
						if r == ')' {
							openParens = false
						}
						continue
					}
					if r == '(' {
						openParens = true
						continue
					}
					withoutBrackets = withoutBrackets + string(r)
				}

				children := make([]string, 0)
				adjacentSiblings := make([]string, 0)
				generalSiblings := make([]string, 0)
				decendants := make([]string, 0)

				// Selector groups
				groups := strings.Split(withoutBrackets, ",")
				if len(groups) > 1 {
					errors[err("selector list not allowed")] = struct{}{}
				}

				// Children
				for _, grouping := range groups {
					grouping = strings.TrimSpace(grouping)

					strs := strings.Split(grouping, ">")
					if len(strs) > 1 {
						errors[err("child selector '>' not allowed")] = struct{}{}
					}
					for _, child := range strs {
						children = append(children, child)
					}
				}

				// Adjacent Siblings
				for _, child := range children {
					child = strings.TrimSpace(child)

					strs := strings.Split(child, "+")
					if len(strs) > 1 {
						errors[err("adjacent sibiling selector '+' not allowed")] = struct{}{}
					}
					for _, adjacentSibling := range strs {
						adjacentSiblings = append(adjacentSiblings, adjacentSibling)
					}
				}

				// General Siblings
				for _, adjacentSibling := range adjacentSiblings {
					adjacentSibling = strings.TrimSpace(adjacentSibling)

					strs := strings.Split(adjacentSibling, "~")
					if len(strs) > 1 {
						errors[err("general sibiling selector '~' not allowed")] = struct{}{}
					}
					for _, generalSibling := range strs {
						generalSiblings = append(generalSiblings, generalSibling)
					}
				}

				// Decendants
				for _, generalSibling := range generalSiblings {
					generalSibling = strings.TrimSpace(generalSibling)

					strs := strings.Split(generalSibling, " ")
					if len(strs) > 1 {
						errors[err("decendant selector ' ' not allowed")] = struct{}{}
					}
					for _, decendant := range strs {
						decendants = append(decendants, decendant)
					}
				}

				// Selectors
				for _, decendant := range decendants {
					decendant = strings.TrimSpace(decendant)

					strs := strings.Split(decendant, " ")
					if len(strs) > 1 {
						errors[err("decendant selector ' ' not allowed")] = struct{}{}
					}
					for _, selector := range strs {
						selector = strings.TrimSpace(selector)

						if strings.Contains(selector, "#") {
							errors[err("id selector '#' not allowed")] = struct{}{}
							continue
						}
						if strings.HasPrefix(selector, ".") {
							e := err(fmt.Sprintf("duplicate selector '%v'", selector))
							if duplicateError, ok := possibleDuplicates[selector]; ok {
								errors[duplicateError] = struct{}{}
								errors[e] = struct{}{}
							}
							possibleDuplicates[selector] = e
							continue
						}
						if strings.HasPrefix(selector, ":") {
							continue
						}
						errors[err("element selector not allowed")] = struct{}{}
					}
				}
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}

	// Order the errors
	errorsSlice := make([]string, 0)
	for err := range errors {
		errorsSlice = append(errorsSlice, err)
	}
	sort.Strings(errorsSlice)
	return errorsSlice, nil
}
