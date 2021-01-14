FROM --platform=$BUILDPLATFORM golang:1.14

ARG BUILDPLATFORM
ARG TARGETARCH
ARG TARGETOS

ENV GO111MODULE=on
WORKDIR /go/src/github.com/wish/kops-controller

# Cache dependencies
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . /go/src/github.com/wish/kops-controller/

# Build controller
RUN CGO_ENABLED=0 GOARCH=${TARGETARCH} GOOS=${TARGETOS} go build -o ./kops-controller -a -installsuffix cgo main.go


FROM alpine:3.11
RUN apk --no-cache add ca-certificates
WORKDIR /usr/bin/
COPY --from=0 /go/src/github.com/wish/kops-controller/kops-controller /usr/bin/kops-controller
RUN addgroup -S appgroup && adduser -S appuser -G appgroup --uid 10011
USER 10011
