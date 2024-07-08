# Welcome to Release Service availability monitor
This service generates availability metrics for the Release Service

## Environment variables
### service variables
| Common                | Default             | Sample               |
|-----------------------|---------------------|----------------------|
| HOME                  | _container default_ | "/tmp/mydir"         |
| SERVICE_LISTEN_PORT   | "8080"              | -                    |
| SERVICE_POLL_INTERVAL | "60"                | -                    |
| SERVICE_CHECKS        | ""                  | "github,gitlab,quay" |

### check variables
|         Quay        |       Github    |       Gitlab    | Gitlab files api  |
|---------------------|-----------------|-----------------|-------------------|
| QUAY_USERNAME       | GITHUB_REPO_URL | GITLAB_REPO_URL | GITLAB_REPO_URL   |
| QUAY_PASSWORD       | GITHUB_REVISION | GITLAB_REVISION | GITLAB_REVISION   |
| QUAY_IMAGE_PULLSPEC | GITHUB_TOKEN    | GITLAB_TOKEN    | GITLAB_TOKEN      |
| QUAY_IMAGE_TAGS     | GITLAB_PATH     | GITLAB_PATH     | GITLAB_PATH       |
|                     |                 |                 | GITLAB_PROJECT_ID |
