FROM golang:1.26-alpine AS build
WORKDIR /src
ARG VERSION=dev
ARG COMMIT=unknown
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" -o /out/fleetpulse ./cmd/fleetpulse

FROM nvidia/cuda:13.3.0-base-ubuntu24.04 AS cuda
ENV NVIDIA_VISIBLE_DEVICES=all
ENV NVIDIA_DRIVER_CAPABILITIES=compute,utility
RUN groupadd --system fleetpulse && useradd --system --gid fleetpulse --home-dir /var/lib/fleetpulse --shell /usr/sbin/nologin fleetpulse
COPY --from=build /out/fleetpulse /usr/local/bin/fleetpulse
RUN mkdir -p /etc/fleetpulse /var/lib/fleetpulse && chown -R fleetpulse:fleetpulse /etc/fleetpulse /var/lib/fleetpulse
USER fleetpulse
EXPOSE 35338
VOLUME ["/var/lib/fleetpulse", "/etc/fleetpulse"]
ENTRYPOINT ["/usr/local/bin/fleetpulse"]
CMD ["serve", "-addr", "0.0.0.0:35338", "-auth=true", "-token-file", "/var/lib/fleetpulse/token", "-deployment-target", "docker"]

FROM rocm/dev-ubuntu-24.04:7.2.4-complete AS rocm
RUN groupadd --system fleetpulse && useradd --system --gid fleetpulse --home-dir /var/lib/fleetpulse --shell /usr/sbin/nologin fleetpulse
COPY --from=build /out/fleetpulse /usr/local/bin/fleetpulse
RUN mkdir -p /etc/fleetpulse /var/lib/fleetpulse && chown -R fleetpulse:fleetpulse /etc/fleetpulse /var/lib/fleetpulse
USER fleetpulse
EXPOSE 35338
VOLUME ["/var/lib/fleetpulse", "/etc/fleetpulse"]
ENTRYPOINT ["/usr/local/bin/fleetpulse"]
CMD ["serve", "-addr", "0.0.0.0:35338", "-auth=true", "-token-file", "/var/lib/fleetpulse/token", "-deployment-target", "docker"]

FROM ubuntu:24.04 AS intel-gpu
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates intel-gpu-tools \
  && rm -rf /var/lib/apt/lists/*
RUN groupadd --system fleetpulse && useradd --system --gid fleetpulse --home-dir /var/lib/fleetpulse --shell /usr/sbin/nologin fleetpulse
COPY --from=build /out/fleetpulse /usr/local/bin/fleetpulse
RUN mkdir -p /etc/fleetpulse /var/lib/fleetpulse && chown -R fleetpulse:fleetpulse /etc/fleetpulse /var/lib/fleetpulse
USER fleetpulse
EXPOSE 35338
VOLUME ["/var/lib/fleetpulse", "/etc/fleetpulse"]
ENTRYPOINT ["/usr/local/bin/fleetpulse"]
CMD ["serve", "-addr", "0.0.0.0:35338", "-auth=true", "-token-file", "/var/lib/fleetpulse/token", "-deployment-target", "docker"]

FROM alpine:3.22 AS runtime
RUN addgroup -S fleetpulse && adduser -S -G fleetpulse fleetpulse
COPY --from=build /out/fleetpulse /usr/local/bin/fleetpulse
RUN mkdir -p /etc/fleetpulse /var/lib/fleetpulse && chown -R fleetpulse:fleetpulse /etc/fleetpulse /var/lib/fleetpulse
USER fleetpulse
EXPOSE 35338
VOLUME ["/var/lib/fleetpulse", "/etc/fleetpulse"]
ENTRYPOINT ["/usr/local/bin/fleetpulse"]
CMD ["serve", "-addr", "0.0.0.0:35338", "-auth=true", "-token-file", "/var/lib/fleetpulse/token", "-deployment-target", "docker"]
