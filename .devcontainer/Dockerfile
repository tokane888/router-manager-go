FROM mcr.microsoft.com/devcontainers/go:2-1.25-trixie

# Go を最新版に更新(devcontainer imageでは1.25までしか公開されていないため)
ARG GO_VERSION=1.26.2
RUN rm -rf /usr/local/go \
    && curl -fsSL https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz -o /tmp/go.tgz \
    && tar -C /usr/local -xzf /tmp/go.tgz \
    && rm /tmp/go.tgz

# install air
RUN curl -sSfL https://raw.githubusercontent.com/cosmtrek/air/master/install.sh | sh -s -- -b /usr/local/bin

# install nodejs, npm
RUN curl -fsSL https://deb.nodesource.com/setup_lts.x | bash - \
    && apt-get update \
    && apt-get install -y nodejs

# install npm packages
RUN npm install -g cc-sdd@latest
RUN npm install -g cspell
RUN npm install -g dprint
RUN npm install -g lefthook
RUN npm install -g @byterover/cipher
RUN npm install -g @goreleaser/goreleaser

# install other packages
RUN apt-get install -y iputils-ping
RUN curl -LO https://github.com/dandavison/delta/releases/download/0.18.2/git-delta_0.18.2_amd64.deb
RUN apt-get install -y ./git-delta_0.18.2_amd64.deb
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.11.4
# /go/pkg/mod/cache/download/ 配下のownerがrootになることを防ぐためvscode user使用
USER vscode
RUN go install mvdan.cc/gofumpt@latest
RUN curl -LsSf https://astral.sh/uv/install.sh | sh
