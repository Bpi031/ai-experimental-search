package ai

import (
    "bufio"
    "context"
    "io"
    "regexp"
    "strings"
    "time"

    gogit "github.com/go-git/go-git/v6"
    "github.com/go-git/go-git/v6/plumbing/object"
)

func keywordSearchRepo(ctx context.Context, repoPath string, opts KeywordSearchOptions) ([]SearchResult, error) {
    repo, err := gogit.PlainOpen(repoPath)
    if err != nil { return nil, err }
    headRef, err := repo.Head()
    if err != nil { return nil, err }
    commit, err := repo.CommitObject(headRef.Hash())
    if err != nil { return nil, err }
    tree, err := commit.Tree()
    if err != nil { return nil, err }

    var re *regexp.Regexp
    var needle string
    if opts.UseRegex {
        flags := 0
        if !opts.CaseSensitive { flags = flags | regexp.IgnoreCase }
        re, err = regexp.CompileOpts(opts.Query, flags)
        if err != nil { return nil, err }
    } else {
        needle = opts.Query
        if !opts.CaseSensitive { needle = strings.ToLower(needle) }
    }

    capK := opts.TopK
    if capK <= 0 { capK = 50 }
    results := make([]SearchResult, 0, capK)
    count := 0

    err = tree.Files().ForEach(func(f *object.File) error {
        if skipPath(f.Name) { return nil }
        r, err := f.Reader()
        if err != nil { return nil }
        defer r.Close()
        lr := io.LimitReader(r, 2*1024*1024)
        scanner := bufio.NewScanner(lr)
        buf := make([]byte, 0, 256*1024)
        scanner.Buffer(buf, 1024*1024)
        lineNo := 1
        for scanner.Scan() {
            line := scanner.Text()
            matched := false
            if opts.UseRegex {
                matched = re.MatchString(line)
            } else {
                hay := line
                if !opts.CaseSensitive { hay = strings.ToLower(hay) }
                matched = strings.Contains(hay, needle)
            }
            if matched {
                chunk := DocumentChunk{
                    ID:         f.Hash.String(),
                    RepoPath:   repoPath,
                    CommitHash: commit.Hash.String(),
                    FilePath:   f.Name,
                    StartLine:  lineNo,
                    EndLine:    lineNo,
                    BlobHash:   f.Hash.String(),
                    Language:   languageFromPath(f.Name),
                    Content:    line + "\n",
                    Meta:       map[string]string{"match": opts.Query},
                    CreatedAt:  time.Now().UTC(),
                }
                // Assign a naive score based on line length (shorter lines rank higher slightly)
                score := 1.0 / float64(1+len(line))
                results = append(results, SearchResult{Chunk: chunk, Score: score})
                count++
                if count >= capK { return nil }
            }
            lineNo++
        }
        return nil
    })
    if err != nil { return nil, err }
    return results, nil
}
