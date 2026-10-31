FROM --platform=$BUILDPLATFORM golang:1.27.3 AS build

WORKDIR /src

COPY go.mod go.sum ./

RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -buildvcs=false -mod=readonly -trimpath -ldflags="-s -w" -o /initializer .

FROM build AS test

RUN CGO_ENABLED=1 go test -race -coverprofile=/coverage.out ./... \
    && sh ci/test-image-tags.sh

FROM gcr.io/distroless/static:nonroot

COPY --from=build --chown=65532:65532 /initializer /initializer

COPY LICENSE-AGPL-3.0 /usr/share/licenses/octelium-initializer/LICENSE-AGPL-3.0

USER 65532:65532

ENTRYPOINT ["/initializer"]
