package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/shlex"
	"github.com/peterrk/slices"
	"golang.org/x/exp/constraints"
)

func ChangeToDirectory(directory string) error {
	err := os.Chdir(directory)
	if err != nil {
		logger.Printf("CommandWithDirectory (Chdir): %s", err.Error())
		return err
	}
	return nil
}

func RunCommand(
	command string,
) (string, error) {
	parsedCommand, err := shlex.Split(command)
	if err != nil {
		logger.Printf("CommandWithDirectory (shlex.Split 1): %s", err.Error())
		return "", err
	}
	if len(parsedCommand) <= 1 {
		err := fmt.Errorf("Command parameter needs arguments passed with it.")
		logger.Printf("CommandWithDirectory (shlex.Split 2): %s", err.Error())
		return "", err
	}
	// git checkout bugs out if there is more than one remote that have the
	// same branches (they probably will be) so those branches need to be
	// locally tracked. if the remote branches that were pulled are not
	// checked out then that when we run into problems
	cmd := exec.Command(parsedCommand[0], parsedCommand[1:]...)
	output, err := cmd.Output()
	if err != nil {
		logger.Printf("CommandWithDirectory (Output): %s", err.Error())
		return "", err
	}
	return string(output), nil
}

func ListBranches(
	directory string,
) ([]string, error) {
	// output, err := RunCommand("git ls-remote --heads origin")
	output, err := RunCommand("git ls-remote --heads .")
	if err != nil {
		logger.Printf("ListBranches (CommandWithDirectory): %s", err.Error())
		return nil, err
	}
	regex, err := regexp.Compile("^[0-9a-f]+\\s*refs/heads/(.*)$")
	if err != nil {
		logger.Printf("ListBranches (Compile): %s", err.Error())
		return nil, err
	}
	newOutput := SplitNewLinePlatformPortable(output)
	lines := []string{}
	for _, line := range newOutput {
		submatch := regex.FindStringSubmatch(line)
		if len(submatch) > 1 {
			lines = append(lines, submatch[1])
		}
	}
	return lines, nil
}

// If this returns an error, the passed in map should be considered volatile
func ContributorCountBranch(
	directory string,
	branch string,
	startTime string,
	endTime string,
	studentCounts map[string]int,
) error {
	_, err := RunCommand(fmt.Sprintf("git checkout %s", branch))
	if err != nil {
		logger.Printf("ContributorCountBranch (CommandWithDirectory 1): %s",
			err.Error())
		return err
	}

	author := ""
	command := fmt.Sprintf("git log %s --since '%s' --until '%s'",
		branch, startTime, endTime)
	command += " --format=COMMIT,%ae,%an --numstat"
	output, err := RunCommand(command)
	if err != nil {
		logger.Printf("ContributorCountBranch (CommandWithDirectory 2): %s",
			err.Error())
		return err
	}

	newOutput := SplitNewLinePlatformPortable(output)

	// TO-DO: Remove lines from log command that became null strings so we
	// don't iterate through them

	firstRegex, err := regexp.Compile("^COMMIT,([^,]*),([^,]*)*$")
	if err != nil {
		logger.Printf("ContributorCountBranch (Compile 1): %s", err.Error())
		return err
	}

	secondRegex, err := regexp.Compile("^\\s*(\\d+)\\s*(\\d+).*$")
	if err != nil {
		logger.Printf("ContributorCountBranch (Compile 2): %s", err.Error())
		return err
	}

	for _, line := range newOutput {
		newLine := strings.TrimSuffix(line, "\n")

		firstGroupMatches := firstRegex.FindStringSubmatch(newLine)
		secondGroupMatches := secondRegex.FindStringSubmatch(newLine)

		if firstRegex.MatchString(newLine) && len(firstGroupMatches) > 2 {
			author = fmt.Sprintf("%s; %s", firstGroupMatches[1],
				firstGroupMatches[2])
		} else if secondRegex.MatchString(newLine) && len(secondGroupMatches) > 2 {
			added, err := strconv.Atoi(secondGroupMatches[1])
			if err != nil {
				logger.Printf("ContributorCountBranch (Atoi 1): %s",
					err.Error())
				return err
			}
			deleted, err := strconv.Atoi(secondGroupMatches[2])
			if err != nil {
				logger.Printf("ContributorCountBranch (Atoi 2): %s",
					err.Error())
				return err
			}
			if author == "" {
				return fmt.Errorf("Author not defined for line: %s", line)
			}
			if val, ok := studentCounts[author]; ok {
				studentCounts[author] = val + added + deleted
			} else {
				studentCounts[author] = 0
			}
		}
	}
	return nil
}

func CountEdits(
	directory string,
	startTime string,
	endTime string,
) (map[string]int, error) {
	_, err := RunCommand("git pull")
	if err != nil {
		logger.Printf("CountEdits (CommandWithDirectory): %s", err.Error())
		return nil, err
	}
	branches, err := ListBranches(directory)
	if err != nil {
		logger.Printf("CountEdits (ListBranches): %s", err.Error())
		return nil, err
	}
	studentCounts := make(map[string]int)
	for idx, branch := range branches {
		err := ContributorCountBranch(directory, branch, startTime, endTime,
			studentCounts)
		if err != nil {
			logger.Printf(
				"CountEdits (ContributorCountBranch) %d: %s", idx, err.Error())
			return nil, err
		}
	}
	return studentCounts, nil
}

func Usage() error {
	var err error = nil
	_, err = fmt.Println("Takes the following params:")
	if err != nil {
		logger.Printf("Usage (Println 1): %s", err.Error())
		return err
	}
	_, err = fmt.Println("-Directory containing repository")
	if err != nil {
		logger.Printf("Usage (Println 2): %s", err.Error())
		return err
	}
	_, err = fmt.Println(
		"-When to start looking at commits, as interpreted by git log's --since")
	if err != nil {
		logger.Printf("Usage (Println 3): %s", err.Error())
		return err
	}
	_, err = fmt.Println(
		"-When to stop looking at commits, as interpreted by git log's --until")
	if err != nil {
		logger.Printf("Usage (Println 4): %s", err.Error())
		return err
	}
	return err
}

// Replace this with the map[string]int key asap because students can have
// different accounts with different names and same emails and visa versa
type student struct {
	email    string
	username string
}

func SplitNewLinePlatformPortable(str string) []string {
	return strings.Split(strings.ReplaceAll(str, "\r\n", "\n"), "\n")
}

func SortedKeysOfMapWithStringKeys[K constraints.Ordered, V any](
	hashmap map[K]V,
) []K {
	sortedKeys := make([]K, 0, len(hashmap))
	for key := range hashmap {
		sortedKeys = append(sortedKeys, key)
	}
	slices.Sort(sortedKeys)
	return sortedKeys
}

func StringSliceOfMapsKeysAndValues[K constraints.Ordered, V any](
	hashmap map[K]V,
) []string {
	sortedKeys := SortedKeysOfMapWithStringKeys(hashmap)
	slice := make([]string, 0, len(hashmap))
	for _, key := range sortedKeys {
		edits := hashmap[key]
		line := fmt.Sprintf("%v: %v", key, edits)
		slice = append(slice, line)
	}
	return slice
}

var logger = log.New(os.Stderr, "", 1)

func main() {
	if len(os.Args) != 4 {
		err := Usage()
		if err != nil {
			logger.Printf("main (Usage): %s", err.Error())
			os.Exit(1)
		}
	} else {
		repository, startTime, endTime := os.Args[1], os.Args[2], os.Args[3]
		err := ChangeToDirectory(repository)
		if err != nil {
			logger.Printf("main (ChangeToDirectory): %s", err.Error())
			os.Exit(1)
		}
		studentCountsMap, err := CountEdits(repository, startTime, endTime)
		if err != nil {
			logger.Printf("main (CountEdits): %s", err.Error())
			os.Exit(1)
		}
		sliceOfStudentEdits := StringSliceOfMapsKeysAndValues(studentCountsMap)
		for idx, studentEdits := range sliceOfStudentEdits {
			_, err := fmt.Println(studentEdits)
			if err != nil {
				logger.Printf("main (StringSliceOfMapsKeysAndValues): %d, %s",
					idx, err.Error())
				os.Exit(1)
			}
		}
	}
}
