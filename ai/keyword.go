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

// keywordSearchRepo is the internal implementation that returns all results in-memory.
// For streaming, use keywordSearchRepoStreaming.
func keywordSearchRepo(ctx context.Context, repoPath string, opts KeywordSearchOptions) ([]SearchResult, error) {
    results := make([]SearchResult, 0, opts.TopK)
    err := keywordSearchRepoStreaming(ctx, repoPath, opts, func(sr SearchResult) error {
        results = append(results, sr)
        return nil
    })
    return results, err
}

// keywordSearchRepoStreaming emits results to a callback as they're found, for VSCode Copilot-style responsiveness.
func keywordSearchRepoStreaming(ctx context.Context, repoPath string, opts KeywordSearchOptions, emit func(SearchResult) error) error {
    repo, err := gogit.PlainOpen(repoPath)
    if err != nil { return err }
    headRef, err := repo.Head()
    if err != nil { return err }
    commit, err := repo.CommitObject(headRef.Hash())
    if err != nil { return err }
    tree, err := commit.Tree()
    if err != nil { return err }

    var re *regexp.Regexp
    var needle string
    if opts.UseRegex {
        pattern := opts.Query
        if !opts.CaseSensitive {
            pattern = "(?i)" + pattern
        }
        re, err = regexp.Compile(pattern)
        if err != nil { return err }
    } else {
        needle = opts.Query
        if !opts.CaseSensitive { needle = strings.ToLower(needle) }
    }

    capK := opts.TopK
    if capK <= 0 { capK = 50 }
    count := 0

    err = tree.Files().ForEach(func(f *object.File) error {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
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
                score := 1.0 / float64(1+len(line))
                if err := emit(SearchResult{Chunk: chunk, Score: score}); err != nil {
                    return err
                }
                count++
                if count >= capK { return io.EOF }
            }
            lineNo++
        }
        return nil
    })
    if err == io.EOF { return nil }
    return err
}
