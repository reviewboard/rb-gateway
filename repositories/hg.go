package repositories

import (
	"fmt"
	"strings"

	hg "bitbucket.org/gohg/gohg"
)

const (
	hgBin = "hg"
)

// A Mercurial repository.
type HgRepository struct {
	RepositoryInfo
}

// Return the name of the repository.
func (repo *HgRepository) GetName() string {
	return repo.Name
}

// Return the path of the repository.
func (repo *HgRepository) GetPath() string {
	return repo.Path
}

// Return the name of the SCM tool.
//
// This will always be `"hg"`.
func (repo *HgRepository) GetScm() string {
	return "hg"
}

// Create a new client for the repository.
//
// The caller is responsible for calling Client.Disconnect() when finished.
func (repo *HgRepository) Client() (*hg.HgClient, error) {
	client := hg.NewHgClient()
	err := client.Connect(hgBin, repo.Path, nil, false)

	if err != nil {
		return nil, err
	}

	return client, nil
}

// Return the contents of the requested file.
//
// On success, it returns the file contents in a byte array. On failure, the
// error will be returned.
func (repo *HgRepository) GetFile(filepath string) ([]byte, error) {
	client, err := repo.Client()
	if err != nil {
		return nil, err
	}
	defer client.Disconnect()
	hgcmd := []string{"cat", filepath}
	return client.ExecCmd(hgcmd)
}

// Return the contents of the requested file at the given changeset.
//
// On success, it returns the file contents in a byte array. On failure, the
// error will be returned.
func (repo *HgRepository) GetFileByCommit(changeset, filepath string) ([]byte, error) {
	client, err := repo.Client()
	if err != nil {
		return nil, err
	}
	defer client.Disconnect()

	hgcmd := []string{"cat", "-r", changeset, filepath}
	return client.ExecCmd(hgcmd)
}

// Return whther or not a file exists.
//
// It returns true if the file exists, false otherwise. On failure, the error
// will also be returned.
func (repo *HgRepository) FileExists(filepath string) (bool, error) {
	client, err := repo.Client()
	if err != nil {
		return false, err
	}
	defer client.Disconnect()

	if _, err = client.ExecCmd([]string{"cat", filepath}); err != nil {
		if isNotExist(err) {
			return false, nil
		} else {
			return false, err
		}
	} else {
		return true, nil
	}
}

// Return whether or not a file exists at a given changeset.
//
// It returns true if the file exists, false otherwise. On failure, the error
// will also be returned.
func (repo *HgRepository) FileExistsByCommit(changeset, filepath string) (bool, error) {
	client, err := repo.Client()
	if err != nil {
		return false, err
	}
	defer client.Disconnect()

	_, err = client.ExecCmd([]string{
		"cat",
		"-r", changeset,
		"--template", "",
		filepath,
	})
	if err != nil {
		if isNotExist(err) {
			return false, nil
		} else {
			return false, err
		}
	} else {
		return true, nil
	}
}

// Return the branches of the repository.
//
// This returns both Mercurial branches and bookmarks.
//
// On failure, the error will also be returned.
func (repo *HgRepository) GetBranches() ([]Branch, error) {
	client, err := repo.Client()
	if err != nil {
		return nil, err
	}
	defer client.Disconnect()

	output, err := client.ExecCmd([]string{
		"branches",
		"--template", "{branch}\\x1f{node}\\x1e",
	})
	if err != nil {
		return nil, err
	}

	branchRecords := strings.Split(strings.TrimRight(string(output), "\x1e"), "\x1e")

	output, err = client.ExecCmd([]string{
		"bookmarks",
		"--template", "{bookmark}\\x1f{node}\\x1e",
	})
	if err != nil {
		return nil, err
	}

	bookmarkRecords := strings.Split(strings.TrimRight(string(output), "\x1e"), "\x1e")
	branches := make([]Branch, 0, len(bookmarkRecords)+len(branchRecords))

	for _, records := range [][]string{branchRecords, bookmarkRecords} {
		for _, record := range records {
			fields := strings.Split(record, "\x1f")
			branches = append(branches, Branch{
				Name: fields[0],
				Id:   fields[1],
			})
		}
	}

	return branches, nil
}

// Return commits from a given starting point.
//
// If `start` is non-empty, that will be used as the starting point. Otherwise
// `branch` will be used.
//
// On failure, the error will also be returned.
func (repo *HgRepository) GetCommits(branch string, start string) ([]CommitInfo, error) {
	if start == "" {
		start = branch
	}

	records, err := repo.Log(nil,
		[]string{
			"{author}",
			"{node}",
			"{date|rfc3339date}",
			"{desc}",
			"{p1node}",
		},
		[]string{start},
		"--follow",
		"--limit", fmt.Sprintf("%d", commitsPageSize),
	)

	if err != nil {
		return nil, err
	}

	commits := make([]CommitInfo, 0, len(records))
	for _, record := range records {
		commits = append(commits, CommitInfo{
			Author:   record[0],
			Id:       record[1],
			Date:     record[2],
			Message:  record[3],
			ParentId: record[4],
		})
	}

	return commits, nil
}

// Return a commit and its diff.
//
// On failure, the error will also be returned.
func (repo *HgRepository) GetCommit(commitId string) (*Commit, error) {
	client, err := repo.Client()
	if err != nil {
		return nil, err
	}
	defer client.Disconnect()

	records, err := repo.Log(client,
		[]string{
			"{author}",
			"{node}",
			"{date|rfc3339date}",
			"{desc}",
			"{p1node}",
		},
		[]string{commitId},
		"--follow",
		"--limit", fmt.Sprintf("%d", commitsPageSize),
	)

	if err != nil {
		return nil, err
	}

	diff, err := client.ExecCmd([]string{
		"diff",
		"--git",
		"--rev", fmt.Sprintf("%s^:%s", commitId, commitId),
	})

	record := records[0]
	commit := Commit{
		CommitInfo: CommitInfo{
			Author:   record[0],
			Id:       record[1],
			Date:     record[2],
			Message:  record[3],
			ParentId: record[4],
		},
		Diff: string(diff),
	}

	return &commit, nil
}

// A convencience method for calling `hg log` and extracting the results.
//
// `client` may be nil, in which case a client will be allocated for the call
// to log that will be deallocated once the emthod finishes. Otherwise, the
// provided client will be used.
//
// `fields` is a list of template parameters. They will be used to generate the
// `--template` argument to `hg log`. [Details on templating in Mercurial][1].
//
// The returned list is a list of the values corresponding the to the template
// parameters in `fields` for each revision in `revisions`.
//
// [1]: https://www.mercurial-scm.org/repo/hg/help/templates
func (repo *HgRepository) Log(client *hg.HgClient, fields, revisions []string, args ...string) ([][]string, error) {
	nFields := len(fields)
	if nFields == 0 {
		return nil, nil
	}

	if client == nil {
		var err error
		client, err = repo.Client()
		if err != nil {
			return nil, err
		}
		defer client.Disconnect()
	}

	format := fmt.Sprintf("%s\\x1e", strings.Join(fields, "\\x1f"))

	command := make([]string, 0, 3+2*len(revisions)+len(args))
	command = append(command, "log", "--template", format)
	for _, rev := range revisions {
		command = append(command, "-r", rev)
	}
	command = append(command, args...)

	output, err := client.ExecCmd(command)
	if err != nil {
		return nil, err
	}

	records := make([][]string, 0, len(revisions))
	rawRecords := strings.Split(strings.TrimRight(string(output), "\x1e"), "\x1e")
	for _, rawRecord := range rawRecords {
		records = append(records, strings.Split(rawRecord, "\x1f"))
	}

	return records, nil
}

func isNotExist(err error) bool {
	return strings.Index(err.Error(), ": no such file in rev ") != -1
}
