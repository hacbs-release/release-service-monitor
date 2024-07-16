# Welcome to Release Service availability monitor
This service generates availability metrics for the Release Service

## Configuration

#### service-config.yaml
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
  http:
    - name: httpcheck
      url: https://www.google.com/robots.txt
      insecure: true
  quay:
    - name: quay-io
      tags:
        - list of tags to check
      pullspec: my-quay-pull-spec
      username: my-quay-robot-account
      password: my-quay-password
```

#### running it
```
./metrics-server service-config.yaml
```

## Config parameters


### Service
| parameter        | default |
| :--              |  :--:   |
| *listen_port*    | 8080    |
| *pool_interval*  | 60      |
| *metrics_previx* | metrics_server |

### Checks
#### GIT
| git | description | example |
| :-- |  --  | -- |
| *name* | check name | mycheck |
| *url* | git repo url | https://github.com/myrepo.git |
| *revision* | git revision | mybranch |
| *path* | file path on git | myfile.txt |
| *token* | git token| mytoken |

#### HTTP
| git | description | example |
| :-- |  --  | -- |
| *name* | check name | mycheck |
| url | url to check | https://www.google.com/robots.txt |
| username | username for `Basic` auth | myuser |
| password | password for `Basic` auth | mypass |
| cert | base64 data TLS cert | - |
| key | base64 data TLS key | - |
| insecure | ignore tls errors | false |
| follow | follow redirects | true |

#### QUAY
| git | description | example |
| :-- |  --  | -- |
| *name* | check name | mycheck |
| *pullspec* | quay.io pullspec | https://quay.io/user/image:tag |
| *username* | quay username  | myuser |
| *password* | quay password | mypass |

## Handling sensitive data

Although it is possible to set the tokens, certs and passwords in the main configuration file, it is recommended
to use special variables to better secure the credentials.

The environment variable naming convention searched by the application is: `<CHECK_NAME>_<SPECIAL_VARIABLE_NAME>`

Example:

For a *git* check named as `mycheck`, the token variable should be exported as `MYCHECK_GIT_TOKEN`.

### Available sensitive env variables

| git        | http           | quay          |
| :-:        | :-:            | :-:           |
| GIT_TOKEN  | HTTP_USERNAME  | QUAY_USERNAME |
|      -     | HTTP_PASSWORD | QUAY_PASSWORD |
|      -     | HTTP_CERT ||
|      -     | HTTP_KEY ||
