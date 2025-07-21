FROM mcr.microsoft.com/devcontainers/go:1-1.24-bookworm

# install air
RUN curl -sSfL https://raw.githubusercontent.com/cosmtrek/air/master/install.sh | sh -s -- -b /usr/local/bin

# install nodejs, npm, yamllint, and notification tools
RUN curl -fsSL https://deb.nodesource.com/setup_lts.x | bash - \
    && apt-get update \
    && apt-get install -y nodejs yamllint libnotify-bin

# install npm packages
RUN npm install -g dprint
RUN npm install -g lefthook

# install other packages
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.1.6
# /go/pkg/mod/cache/download/ 配下のownerがrootになることを防ぐためvscode user使用
USER vscode
RUN go install mvdan.cc/gofumpt@latest
