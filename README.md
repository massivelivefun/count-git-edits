# count-git-edits

A quick and dirty Go binary to count the number of line additions/subtractions per contributor.
Considers all remote branches, not just `main` or `master`.
Considers each contributer to be a unique name/email combination.

## Usage

```console
go run count_git_edits.go <<repository_location>> <<starting_commit_point>> <<ending_commit_point>>
```

...where:

- `<<repository_location>>` is a path to a repository.
- `<<starting_commit_point>>` specifies a starting time when the start looking for commits.
    This is passed directory to the `--since` option of `git log`; see details about possibilities in `man gitrevisions`.
- `<<ending_commit_point>>` specifies a starting time when to stop looking for commits.
    This is passed directory to the `--until` option of `git log`; see details about possibilities in `man gitrevisions`.

## Problems

Some users in the count may have the same name but different email for their accounts.
The visa versa may also be the case.
Accounts with the same email and name should be combined in the map as a well defined key.
This way users with multiple accounts can be considered as one entity in the count.

## Based off of...

[count_git_edits](https://github.com/kyledewey/count_git_edits)
