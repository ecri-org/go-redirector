FROM golang:alpine as base
RUN adduser -D -u 1000 runner

FROM scratch
COPY --from=base /etc/passwd /etc/passwd
USER runner

# Copy app
COPY dist/linux/go-redirector-linux-amd64 /go-redirector
COPY views /views
COPY redirect-map.yml /

CMD ["/go-redirector", "run"]
