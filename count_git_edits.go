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
	strings, err := shlex.Split(command)
	if err != nil {
		logger.Printf("CommandWithDirectory (shlex.Split 1): %s", err.Error())
		return "", err
	}
	if len(strings) <= 1 {
		err := fmt.Errorf("Command parameter needs arguments passed with it.")
		logger.Printf("CommandWithDirectory (shlex.Split 2): %s", err.Error())
		return "", err
	}
	// git checkout bugs out if there is more than one remote that have the
	// same branches (they probably will) so it needs to be locally tracked
	// if the remote branches that were pulled are not checked out then
	// that when we run into problems
	cmd := exec.Command(strings[0], strings[1:]...)
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
	_, err := RunCommand(fmt.Sprintf("git checkout %s", branch))
	if err != nil {
		logger.Printf("ContributorCountBranch (CommandWithDirectory 1): %s",
			err.Error())
		return err
	}

	author := ""
	command := fmt.Sprintf("git log %s --since '%s' --until '%s'",
		branch, start_time, end_time)
	command += " --format=COMMIT,%ae,%an --numstat"
	output, err := RunCommand(command)
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
		err := ChangeToDirectory(os.Args[1])
		if err != nil {
			logger.Printf("main (ChangeToDirectory): %s", err.Error())
			os.Exit(1)
		}
		counts_map, err := CountEdits(os.Args[1], os.Args[2], os.Args[3])
		if err != nil {
			logger.Printf("main (CountEdits): %s", err.Error())
			os.Exit(1)
		}
		slice_of_students_edits := StringSliceOfMapsKeysAndValues(counts_map)
		for idx, student_edits := range slice_of_students_edits {
			_, err := fmt.Println(student_edits)
			if err != nil {
				logger.Printf("main (StringSliceOfMapsKeysAndValues): %d, %s",
					idx, err.Error())
				os.Exit(1)
			}
		}
	}
}
