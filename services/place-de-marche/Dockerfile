FROM golang:1.26.10-bookworm AS construction
ENV CGO_ENABLED=0 \
    GOWORK=off \
    GOFLAGS=-mod=readonly \
    GOTOOLCHAIN=local
WORKDIR /src
COPY libs/platform/go.mod libs/platform/go.sum libs/platform/
COPY services/place-de-marche/go.mod services/place-de-marche/go.sum services/place-de-marche/
WORKDIR /src/services/place-de-marche
RUN go mod download && go mod verify
WORKDIR /src
COPY libs/platform/ libs/platform/
COPY services/place-de-marche/ services/place-de-marche/
WORKDIR /src/services/place-de-marche
RUN go build -trimpath -ldflags="-s -w" -o /out/place-de-marche ./cmd/place-de-marche

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=construction /out/place-de-marche /place-de-marche
USER 65532:65532
EXPOSE 8080 9090
ENTRYPOINT ["/place-de-marche"]
CMD ["serve"]
