package ai_test

import (
    "os"
    "time"

    git "github.com/go-git/go-git/v6"
    "github.com/go-git/go-git/v6/plumbing/object"
)

func mustWrite(t interface{ Fatal(args ...any) }, path string, data string) {
    if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
        t.Fatal(err)
    }
}

func mustCommitAll(t interface{ Fatal(args ...any) }, repo *git.Repository, msg string) {
    w, err := repo.Worktree()
    if err != nil { t.Fatal(err) }
    if err := w.AddWithOptions(&git.AddOptions{All: true}); err != nil { t.Fatal(err) }
    _, err = w.Commit(msg, &git.CommitOptions{
        Author: &object.Signature{Name: "AI Tester", Email: "ai@test", When: time.Now()},
    })
    if err != nil { t.Fatal(err) }
}
