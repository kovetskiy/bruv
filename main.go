package main

import (
	"bufio"
	"crypto/sha256"
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
  -h --help         Show this screen.
  --version         Show version.
`
)

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

		err = showDiff(cacheDir, hash, url, src, dst)
		if err != nil {
			log.Fatal(
				karma.Format(
					err,
					"unable to show difference for remote: %s", url,
				),
			)
		}
	}
}

func showDiff(dir string, remote string, url, src, dst string) error {
	stdout, _, err := executil.Run(
		exec.Command(
			"git", "-C", dir,
			"rev-list", "--left-right", "--count",
			remote+"/"+src+"..."+remote+"/"+dst,
		),
	)
	if err != nil {
		return err
	}

	parts := strings.Split(strings.TrimSpace(string(stdout)), "\t")
	if len(parts) != 2 {
		return errors.New(
			"unexpected output of git rev-list, expected 2 parts around \\t",
		)
	}

	behind, err := strconv.Atoi(parts[0])
	if err != nil {
		return karma.Format(
			err,
			"unable to examine output of rev-list: %s",
			parts[0],
		)
	}

	ahead, err := strconv.Atoi(parts[1])
	if err != nil {
		return karma.Format(
			err,
			"unable to examine output of rev-list: %s",
			parts[1],
		)
	}

	if behind == 0 && ahead == 0 {
		fmt.Printf("%s\t%s is same as %s\n", url, dst, src)
	} else {
		message := []string{}
		if ahead > 0 {
			message = append(message, fmt.Sprintf("%d commits ahead", ahead))
		}

		if behind > 0 {
			message = append(message, fmt.Sprintf("%d commits behind", behind))
		}

		fmt.Printf(
			"%s\tcompared to %s, %s is %s\n",
			url,
			src,
			dst,
			strings.Join(message, " and "),
		)
	}

	return nil
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
