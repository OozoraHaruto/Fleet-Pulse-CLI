FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/fleetpulse ./cmd/fleetpulse

FROM alpine:3.22
RUN addgroup -S fleetpulse && adduser -S -G fleetpulse fleetpulse
COPY --from=build /out/fleetpulse /usr/local/bin/fleetpulse
RUN mkdir -p /etc/fleetpulse /var/lib/fleetpulse && chown -R fleetpulse:fleetpulse /etc/fleetpulse /var/lib/fleetpulse
USER fleetpulse
EXPOSE 8080
VOLUME ["/var/lib/fleetpulse", "/etc/fleetpulse"]
ENTRYPOINT ["/usr/local/bin/fleetpulse"]
CMD ["serve", "-addr", "0.0.0.0:8080", "-auth=true", "-token-file", "/var/lib/fleetpulse/token", "-deployment-target", "docker"]
