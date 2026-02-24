FROM docker.io/library/golang:1.24-alpine AS build

RUN apk add --no-cache gcc musl-dev

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go generate ./...
RUN go build -ldflags='-s -w -extldflags "-static"' -o /bashistdb .

FROM gcr.io/distroless/static:nonroot

ENV USER=bashistdb
ENV HOME=/data

COPY --from=build /bashistdb /usr/local/bin/bashistdb

VOLUME /data
EXPOSE 25625

ENTRYPOINT ["bashistdb"]
CMD ["-server"]
