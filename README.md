# ensure-latest-go

Ensure-latest-go is a GitHub Action to keep Dockerfiles, Travis CI configs,
and GitHub Actions using the latest stable version of Go.


This Action is designed to work in conjunction with `actions/checkout` and
`peter-evans/create-pull-request` to keep your repository on the latest released
version of Go. When a new Go release is discovered by it, a new pull request
will be generated with the appropriate changes to your configuration.

## Usage
Below is the common case configuration:

```yaml
name: Keeping Go up to date
on:
  schedule:
    - cron: 47 4 * * *
  push:
    branches:
      - master
jobs:
  fresh_go:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@master
        with:
          ref: master
      - uses: jmhodges/ensure-latest-go@v1.0.1
        id: ensure_go
      - run: echo "##[set-output name=pr_title;]update to latest Go release ${{ steps.ensure_go.outputs.go_version}}"
        id: pr_title_maker
      - name: Create pull request
        uses: peter-evans/create-pull-request@v1.5.2
        env:
          PULL_REQUEST_TITLE: ${{ steps.pr_title_maker.outputs.pr_title }}
          PULL_REQUEST_BODY: Auto-generated pull request created by the GitHub Actions [create-pull-request](https://github.com/peter-evans/create-pull-request) and [ensure-latest-go](https://github.com/jmhodges/ensure-latest-go).
          COMMIT_MESSAGE: ${{ steps.pr_title_maker.outputs.pr_title }}
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          BRANCH_SUFFIX: none
          PULL_REQUEST_BRANCH: ensure-latest-go/patch-${{ steps.ensure_go.outputs.go_version }}
```

To enjoy the full benefits with GitHub Actions, you'll need to add a `.github/versions/go` file to your repository with the version of Go you want `actions/setup-go` to use. Then, modify your `actions/setup-go` job to include:

```yaml
    steps:
    - name: Check out the code
      uses: actions/checkout@v1
    - name: Read Go versions
      run: echo "##[set-output name=go_version;]$(cat .github/versions/go)"
      id: go_versions
    - name: Set up Go
      uses: actions/setup-go@v1
      with:
        go-version: ${{ steps.go_versions.outputs.go_version }}
      id: go

```

That'll get you pretty far. The default configuration documented above will
update any files named `Dockerfile` that use `FROM golang` statements, the
top-level `.travis.yml` file, and any GitHub Action files in
`.github/workflows/` that use `actions/setup-go`. If any of those files don't
exist, they'll just be skipped.

If you'd like more control, there are a few optional arguments you can set with `with` (all file paths are relative to the top-level directory of the repository):

### Inputs

| Name | Description | Default |
| --- | --- | --- |
| exclude | An optional comma-separated list of file paths  of any type that will not be updated.| none |
| dockerfiles | An optional comma-seperated list of Dockerfiles to update when a new Go version is released. If set, it will override the default behavior of updating any files named `Dockerfile` using a `golang` image. | none |
| travisfiles | An optional comma-seperated list of Travis CI config files to update when a new Go version is released. If set, it will override the default behavior of updating (but not creating) the "go" setting in a top-level .travis.yml file. | none |

### Outputs

This project also provides an output variable of the Go version it used in order
to generate clear pull requests. It's used in the common case example workflow
above.

| Name | Description |
| --- | --- |
| go_version | The version of Go used to update the configured files. |
