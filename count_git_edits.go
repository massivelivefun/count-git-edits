package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/peterrk/slices"
	"golang.org/x/exp/constraints"
)

func CommandWithDirectory(
	directory string,
	command string,
) (string, error) {
	err := os.Chdir(directory)
	if err != nil {
		logger.Printf("CommandWithDirectory (Chdir):%s", err.Error())
		return "", err
	}

	// Command parameter is split on spaces
	strings := strings.Fields(command)
	if len(strings) <= 1 {
		err := fmt.Errorf("Command parameter needs arguments passed with it.")
		logger.Printf("CommandWithDirectory (Fields): %s", err.Error())
		return "", err
	}

	// This whole conditional needs to be cleaned up
	if strings[1] == "log" {
		// Can't really build the arguments and assume that its always dates
		// Especially when the date could be something like "Today"
		// first_date := fmt.Sprintf("%s %s %s",
		// 	strings[4], strings[5], strings[6])
		// second_date := fmt.Sprintf("%s %s %s",
		// 	strings[8], strings[9], strings[10])
		// Hacky fix but it honestly works and covers most obvious cases
		flags := make([]string, 0, 8)
		flags = append(flags, strings[1])
		flags = append(flags, strings[2])
		flags = append(flags, strings[3])
		flags = append(flags, os.Args[2])
		flags = append(flags, "--until")
		flags = append(flags, os.Args[3])
		flags = append(flags, "--format=COMMIT,%ae,%an")
		flags = append(flags, "--numstat")
		cmd := exec.Command(strings[0], flags...)
		output, err := cmd.Output()
		if err != nil {
			logger.Printf("CommandWithDirectory (Output 1): %s", err.Error())
			return "", err
		}
		return string(output), nil
	} else {
		cmd := exec.Command(strings[0], strings[1:]...)
		output, err := cmd.Output()
		if err != nil {
			logger.Printf("CommandWithDirectory (Output 2): %s", err.Error())
			return "", err
		}
		return string(output), nil
	}
}

func ListBranches(directory string) ([]string, error) {
	output, err := CommandWithDirectory(directory,
		"git ls-remote --heads origin")
	if err != nil {
		logger.Printf("ListBranches (CommandWithDirectory): %s", err.Error())
		return nil, err
	}
	regex, err := regexp.Compile("^[0-9a-f]+\\s*refs/heads/(.*)$")
	if err != nil {
		logger.Printf("ListBranches (Compile): %s", err.Error())
		return nil, err
	}
	new_output := SplitNewLinePlatformPortable(output)
	lines := []string{}
	for _, line := range new_output {
		submatch := regex.FindStringSubmatch(line)
		if submatch == nil {
			continue
		}
		lines = append(lines, submatch[1])
	}
	return lines, nil
}

// If this returns an error, the passed in map should be considered volatile
func ContributorCountBranch(
	directory string,
	branch string,
	start_time string,
	end_time string,
	student_counts map[string]int,
) error {
	_, err := CommandWithDirectory(directory, fmt.Sprintf("git checkout %s",
		branch))
	if err != nil {
		logger.Printf("ContributorCountBranch (CommandWithDirectory 1): %s",
			err.Error())
		return err
	}

	author := ""
	command := fmt.Sprintf("git log %s --since '%s' --until '%s'",
		branch, start_time, end_time)
	command += " --format=COMMIT,%ae,%an --numstat"
	output, err := CommandWithDirectory(directory, command)
	if err != nil {
		logger.Printf("ContributorCountBranch (CommandWithDirectory 2): %s",
			err.Error())
		return err
	}

	new_output := SplitNewLinePlatformPortable(output)

	// TO-DO: Remove lines from log command that became null strings so we
	// don't iterate through them

	first_regex, err := regexp.Compile("^COMMIT,([^,]*),([^,]*)*$")
	if err != nil {
		logger.Printf("ContributorCountBranch (Compile 1): %s", err.Error())
		return err
	}

	second_regex, err := regexp.Compile("^\\s*(\\d+)\\s*(\\d+).*$")
	if err != nil {
		logger.Printf("ContributorCountBranch (Compile 2): %s", err.Error())
		return err
	}

	for _, line := range new_output {
		new_line := strings.TrimSuffix(line, "\n")

		first_group_matches := first_regex.FindStringSubmatch(new_line)
		second_group_matches := second_regex.FindStringSubmatch(new_line)

		if first_regex.MatchString(new_line) && len(first_group_matches) > 2 {
			author = fmt.Sprintf("%s; %s", first_group_matches[1],
				first_group_matches[2])
		} else if second_regex.MatchString(new_line) && len(second_group_matches) > 2 {
			added, err := strconv.Atoi(second_group_matches[1])
			if err != nil {
				logger.Printf("ContributorCountBranch (Atoi 1): %s",
					err.Error())
				return err
			}
			deleted, err := strconv.Atoi(second_group_matches[2])
			if err != nil {
				logger.Printf("ContributorCountBranch (Atoi 2): %s",
					err.Error())
				return err
			}
			if author == "" {
				return fmt.Errorf("Author not defined for line: %s", line)
			}
			if val, ok := student_counts[author]; ok {
				student_counts[author] = val + added + deleted
			} else {
				student_counts[author] = 0
			}
		}
	}
	return nil
}

func CountEdits(
	directory string,
	start_time string,
	end_time string,
) (map[string]int, error) {
	_, err := CommandWithDirectory(directory, "git pull")
	if err != nil {
		logger.Printf("CountEdits (CommandWithDirectory): %s", err.Error())
		return nil, err
	}
	branches, err := ListBranches(directory)
	if err != nil {
		logger.Printf("CountEdits (ListBranches): %s", err.Error())
		return nil, err
	}
	student_counts := make(map[string]int)
	for idx, branch := range branches {
		err := ContributorCountBranch(directory, branch, start_time, end_time,
			student_counts)
		if err != nil {
			logger.Printf(
				"CountEdits (ContributorCountBranch) %d: %s", idx, err.Error())
			return nil, err
		}
	}
	return student_counts, nil
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
	return nil
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
	the_map map[K]V,
) []K {
	sorted_keys := make([]K, 0, len(the_map))
	for key := range the_map {
		sorted_keys = append(sorted_keys, key)
	}
	slices.Sort(sorted_keys)
	return sorted_keys
}

func StringSliceOfMapsKeysAndValues[K constraints.Ordered, V any](
	the_map map[K]V,
) []string {
	sorted_keys := SortedKeysOfMapWithStringKeys(the_map)
	slice := make([]string, 0, len(the_map))
	for _, key := range sorted_keys {
		edits := the_map[key]
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
		counts_map, err := CountEdits(os.Args[1], os.Args[2], os.Args[3])
		if err != nil {
			logger.Printf("main (CountEdits): %s", err.Error())
			os.Exit(1)
		}
		slice_of_students_edits := StringSliceOfMapsKeysAndValues(counts_map)
		for _, student_edits := range slice_of_students_edits {
			fmt.Println(student_edits)
		}
	}
}
