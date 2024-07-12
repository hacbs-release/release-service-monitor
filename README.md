# Welcome to Release Service availability monitor
This service generates availability metrics for the Release Service

## Configuration example
### service-config.yaml
```
---
service:
  listen_port: 8080
  pool_interval: 300
  metrics_prefix: my_prefix
checks:
  git:
    - name: github
      url: my-github-repository-url
      revision: my-github-branch
      path: path-to-my-file-on-github
      token: my-token
    - name: gitlab
      url: my-gitlab-repository-url
      revision: my-gitlab-branch
      path: path-to-my-file-on-gitlab
      token: my-token
  quay:
    - name: quay-io
      tags:
        - list of tags to check
      pullspec: my-quay-pull-spec
      username: my-quay-robot-account
      password: my-quay-password
```

To run it
```
./metrics-server service-config.yaml
```
## Handling passwords

Although it is possible to set the tokens and passwords in the main configuration file, it is also possible
to use special variables to better secure the credentials.

The environment variable format is `<CHECK NAME>_GIT_TOKEN` for `git` and `<CHECK_NAME>_QUAY_PASSWORD` for
`quay` checks.

Example:

A *git* check named `github` can have its token set through the `GITHUB_GIT_TOKEN` env variable.
