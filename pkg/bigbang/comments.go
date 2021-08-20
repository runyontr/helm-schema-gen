package bigbang

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"regexp"

	"github.com/norwoodj/helm-docs/pkg/helm"
)

var valuesDescriptionRegex = regexp.MustCompile("^\\s*#\\s*(.*)\\s+--\\s*(.*)$")
var commentContinuationRegex = regexp.MustCompile("^\\s*# (.*)$")
var defaultValueRegex = regexp.MustCompile("^\\s*# @default -- (.*)$")

func parseChartValuesFileComments(chartDirectory string) (map[string]helm.ChartValueDescription, error) {

	var fullChartSearchRoot string

	if path.IsAbs(chartDirectory) {
		fullChartSearchRoot = chartDirectory
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error getting working directory: %s\n", err)
			return map[string]helm.ChartValueDescription{}, err
		}

		fullChartSearchRoot = path.Join(cwd, chartDirectory)
	}

	valuesPath := path.Join(fullChartSearchRoot, "values.yaml")
	valuesFile, err := os.Open(valuesPath)

	if err != nil {
		fmt.Printf("error opening values file: %v\n", err)
		return map[string]helm.ChartValueDescription{}, err
	}

	defer valuesFile.Close()

	keyToDescriptions := make(map[string]helm.ChartValueDescription)
	scanner := bufio.NewScanner(valuesFile)
	foundValuesComment := false
	commentLines := make([]string, 0)

	for scanner.Scan() {
		currentLine := scanner.Text()
		fmt.Printf("current line: %v\n", currentLine)
		// If we've not yet found a values comment with a key name, try and find one on each line
		if !foundValuesComment {
			match := valuesDescriptionRegex.FindStringSubmatch(currentLine)
			fmt.Printf("Found %v matches\n", match)
			// if len(match) < 3 {
			// 	fmt.Printf("Less than 3 matches, so continuing\n")
			// 	continue
			// }
			if match[1] == "" {
				fmt.Printf("Empty Match, so continuing\n")
				continue
			}

			foundValuesComment = true
			commentLines = append(commentLines, currentLine)
			fmt.Printf("Added line to current comment block\n")
			continue
		}

		// If we've already found a values comment, on the next line try and parse a custom default value. If we find one
		// that completes parsing for this key, add it to the list and reset to searching for a new key
		defaultCommentMatch := defaultValueRegex.FindStringSubmatch(currentLine)
		commentContinuationMatch := commentContinuationRegex.FindStringSubmatch(currentLine)

		if len(defaultCommentMatch) > 1 || len(commentContinuationMatch) > 1 {
			commentLines = append(commentLines, currentLine)
			continue
		}

		// If we haven't continued by this point, we didn't match any of the comment formats we want, so we need to add
		// the in progress value to the map, and reset to looking for a new key
		key, description := ParseComment(commentLines)
		keyToDescriptions[key] = description
		commentLines = make([]string, 0)
		foundValuesComment = false
	}

	return keyToDescriptions, nil
}

func ParseComment(commentLines []string) (string, helm.ChartValueDescription) {
	var valueKey string
	var c helm.ChartValueDescription
	var docStartIdx int

	for i := range commentLines {
		match := valuesDescriptionRegex.FindStringSubmatch(commentLines[i])
		if len(match) < 3 {
			continue
		}

		valueKey = match[1]
		c.Description = match[2]
		docStartIdx = i
		break
	}

	for _, line := range commentLines[docStartIdx+1:] {
		defaultCommentMatch := defaultValueRegex.FindStringSubmatch(line)

		if len(defaultCommentMatch) > 1 {
			c.Default = defaultCommentMatch[1]
			continue
		}

		commentContinuationMatch := commentContinuationRegex.FindStringSubmatch(line)

		if len(commentContinuationMatch) > 1 {
			c.Description += " " + commentContinuationMatch[1]
			continue
		}
	}

	return valueKey, c
}
