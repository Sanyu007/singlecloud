FROM golang:1.12.5-alpine3.9 AS build

ARG version
ARG buildtime

RUN mkdir -p /go/src/github.com/zdnscloud/singlecloud
COPY . /go/src/github.com/zdnscloud/singlecloud
WORKDIR /go/src/github.com/zdnscloud/singlecloud
RUN cp zke_image.yml vendor/github.com/zdnscloud/zke/image_config.yml
RUN go generate vendor/github.com/zdnscloud/zke/types/generate.go
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-w -s -X main.version=$version -X main.build=$buildtime -X github.com/zdnscloud/singlecloud/pkg/zke.singleCloudVersion=$version" cmd/singlecloud/singlecloud.go


FROM scratch
COPY --from=build /go/src/github.com/zdnscloud/singlecloud/singlecloud /
ENTRYPOINT ["/singlecloud"]
