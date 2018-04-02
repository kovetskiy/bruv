package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/docopt/docopt-go"
	"github.com/reconquest/executil-go"
	"github.com/reconquest/karma-go"
)

var (
	version = "[manual build]"
	usage   = "bruv " + version + `


Usage:
  bruv [options] <src> <dst> <url>...
  bruv [options] <src> <dst> -i
  bruv -h | --help
  bruv --version

Options:
  -i --stdin        Use stdin as list of repositories.
  -c --cache <dir>  Use this directory for cache.
                     [default: $HOME/.cache/bruv/]
  -j --json         Output in JSON.
  -h --help         Show this screen.
  --version         Show version.
`
)

type status struct {
	URL     string   `json:"url"`
	Equal   bool     `json:"equal"`
	Status  string   `json:"status"`
	Commits []string `json:"commits"`
}

func main() {
	args, err := docopt.Parse(os.ExpandEnv(usage), nil, true, version, false)
	if err != nil {
		panic(err)
	}

	var (
		urls     = args["<url>"].([]string)
		src      = args["<src>"].(string)
		dst      = args["<dst>"].(string)
		cacheDir = args["--cache"].(string)
		useJSON  = args["--json"].(bool)
	)

	if args["--stdin"].(bool) {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) != 0 {
				urls = append(urls, line)
			}
		}
	}

	err = initCache(cacheDir)
	if err != nil {
		log.Fatal(karma.Format(err, "unable to init cache dir"))
	}

	statuses := []status{}
	space := getLongest(urls)

	for _, url := range urls {
		hash := getHash(url)

		exists, err := isRemoteExists(cacheDir, hash)
		if err != nil {
			log.Fatal(
				karma.Format(
					err,
					"unable to determine state of git remote: %s",
					url,
				),
			)
		}

		if !exists {
			err = initRemote(cacheDir, hash, url)
			if err != nil {
				log.Fatal(
					karma.Format(
						err,
						"unable to init remote: %s",
						url,
					),
				)
			}
		}

		err = updateRemote(cacheDir, hash)
		if err != nil {
			log.Fatal(
				karma.Format(err, "unable to update remote: %s", url),
			)
		}

		result, err := getStatus(cacheDir, hash, url, src, dst)
		if err != nil {
			log.Fatal(
				karma.Format(
					err,
					"unable to show difference for remote: %s", url,
				),
			)
		}

		if useJSON {
			statuses = append(statuses, result)
		} else {
			fmt.Printf(
				"%-"+fmt.Sprint(space)+"s %s\n",
				result.URL,
				result.Status,
			)
			if !result.Equal {
				for _, commit := range result.Commits {
					fmt.Println(" ", commit)
				}
			}
		}
	}

	if useJSON {
		contents, err := json.MarshalIndent(statuses, "", "  ")
		if err != nil {
			log.Fatal(karma.Format(err, "unable to marshal to JSON"))
		}

		fmt.Println(string(contents))
	}
}

func getLongest(items []string) int {
	longest := 0
	for _, item := range items {
		length := len(item)
		if length > longest {
			longest = length
		}
	}

	return longest
}

func getStatus(
	dir string,
	remote string,
	url, src, dst string,
) (status, error) {
	stdout, _, err := executil.Run(
		exec.Command(
			"git", "-C", dir,
			"rev-list", "--left-right", "--count",
			remote+"/"+src+"..."+remote+"/"+dst,
		),
	)
	if err != nil {
		return status{}, err
	}

	parts := strings.Split(strings.TrimSpace(string(stdout)), "\t")
	if len(parts) != 2 {
		return status{}, errors.New(
			"unexpected output of git rev-list, expected 2 parts around \\t",
		)
	}

	behind, err := strconv.Atoi(parts[0])
	if err != nil {
		return status{}, karma.Format(
			err,
			"unable to examine output of rev-list: %s",
			parts[0],
		)
	}

	ahead, err := strconv.Atoi(parts[1])
	if err != nil {
		return status{}, karma.Format(
			err,
			"unable to examine output of rev-list: %s",
			parts[1],
		)
	}

	var result status
	result.URL = url

	if behind == 0 && ahead == 0 {
		result.Status = fmt.Sprintf("%s is same as %s", dst, src)
		result.Equal = true
	} else {
		message := []string{}
		if ahead > 0 {
			message = append(message, fmt.Sprintf("%d commits ahead", ahead))
		}

		if behind > 0 {
			message = append(message, fmt.Sprintf("%d commits behind", behind))
		}

		result.Status = fmt.Sprintf(
			"compared to %s, %s is %s",
			src,
			dst,
			strings.Join(message, " and "),
		)

		commits, err := getLogs(dir, remote, src, dst)
		if err != nil {
			return status{}, karma.Format(
				err,
				"unable to get git logs",
			)
		}

		result.Commits = commits
	}

	return result, nil
}

func getLogs(dir string, remote string, src, dst string) ([]string, error) {
	stdout, _, err := executil.Run(
		exec.Command(
			"git", "-C", dir,
			"log", "--oneline", "--left-right",
			remote+"/"+src+"..."+remote+"/"+dst,
		),
	)
	if err != nil {
		return nil, err
	}

	return strings.Split(strings.TrimSpace(string(stdout)), "\n"), nil
}

func updateRemote(dir string, remote string) error {
	_, _, err := executil.Run(
		exec.Command("git", "-C", dir, "remote", "update", remote),
	)
	if err != nil {
		return err
	}

	return nil
}

func initRemote(dir string, remote string, url string) error {
	_, _, err := executil.Run(
		exec.Command("git", "-C", dir, "remote", "add", remote, url),
	)
	if err != nil {
		return err
	}

	return nil
}

func isRemoteExists(dir string, remote string) (bool, error) {
	stdout, _, err := executil.Run(
		exec.Command("git", "-C", dir, "remote", "show", "-n"),
	)
	if err != nil {
		return false, err
	}

	for _, item := range strings.Split(string(stdout), "\n") {
		if item == remote {
			return true, nil
		}
	}

	return false, nil
}

func getHash(target string) string {
	hasher := sha256.New()
	_, err := hasher.Write([]byte(target))
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func initCache(dir string) error {
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Open(gitDir); err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}

		_, _, err := executil.Run(exec.Command("git", "-C", dir, "init"))
		if err != nil {
			return err
		}
	}

	return nil
}
