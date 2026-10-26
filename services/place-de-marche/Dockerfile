FROM golang:1.26.10-bookworm@sha256:632dff7351799a5a9954a48dc06670ade72ee44e8c351cdc0e5a57af2fea21af AS construction
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

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab
COPY --from=construction /out/place-de-marche /place-de-marche
USER 65532:65532
EXPOSE 8080 9090
ENTRYPOINT ["/place-de-marche"]
CMD ["serve"]
