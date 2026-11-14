FROM golang:1.26.3-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=0
RUN go build -buildvcs=false -trimpath -ldflags="-s -w" -o /out/server ./cmd/server && \
    go build -buildvcs=false -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM build AS test
RUN go test -buildvcs=false ./...

FROM scratch AS runtime
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/server /server
COPY --from=build /out/migrate /migrate
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/server"]
