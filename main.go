package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

var (
	currentRevision = ""
)

type Project struct {
	Name       string
	Repository string
	Path       string
}

func cloneAndOpenRepository(project Project) (*git.Repository, error) {
	os.Setenv("GIT_DIR", "./tmp/git")

	gitDirOut := fmt.Sprintf("%s/%s", os.Getenv("GIT_DIR"), project.Name)

	r, err := git.PlainClone(gitDirOut, false, &git.CloneOptions{
		URL:      project.Repository,
		Progress: os.Stdout,
	})
	if err != nil && err != git.ErrRepositoryAlreadyExists {
		return nil, err
	}

	r, err = git.PlainOpen(gitDirOut)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func getRevisionDiff(r *git.Repository) (string, error) {
	w, err := r.Worktree()
	if err != nil {
		return "", err
	}

	err = w.Pull(&git.PullOptions{RemoteName: "origin"})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return "", err
	}

	h, err := r.ResolveRevision(plumbing.Revision("HEAD"))
	if err != nil {
		return "", nil
	}

	if currentRevision != h.String() {
		currentRevision = h.String()
		return h.String(), nil
	}

	return "", nil
}

func initializeRevision(r *git.Repository) error {
	p, err := r.Head()
	if err != nil {
		return err
	}

	currentRevision = p.Hash().String()
	return nil
}

func sync(r *git.Repository) error {
	d, err := getRevisionDiff(r)
	if err != nil {
		return nil
	}

	if d != "" {
		// TODO: sync logic
		fmt.Println("EVENT")
		return nil
	}

	return nil
}

func main() {
	resync := make(chan bool)
	projects := []Project{
		{
			Name:       "argo-examples",
			Repository: "https://github.com/maycommit/argo-example.git",
		},
	}

	for _, project := range projects {
		r, err := cloneAndOpenRepository(project)
		if err != nil {
			log.Fatalln(err)
		}

		err = initializeRevision(r)
		if err != nil {
			log.Fatalln(err)
		}

		go func() {
			fmt.Println("Start gitops engine...")
			ticker := time.NewTicker(5 * time.Second)
			for {
				select {
				case <-ticker.C:
					err := sync(r)
					if err != nil {
						log.Fatalln(err)
					}
				case <-resync:
					err := sync(r)
					if err != nil {
						log.Fatalln(err)
					}
				}
			}
		}()
	}

	http.HandleFunc("/sync", func(w http.ResponseWriter, r *http.Request) {
		resync <- true
	})

	fmt.Println("Start server on 8080...")
	log.Println(http.ListenAndServe(":8080", nil))
}
