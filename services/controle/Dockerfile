FROM golang:1.26.10-bookworm@sha256:632dff7351799a5a9954a48dc06670ade72ee44e8c351cdc0e5a57af2fea21af AS construction
ENV CGO_ENABLED=0 \
    GOWORK=off \
    GOFLAGS=-mod=readonly \
    GOTOOLCHAIN=local
WORKDIR /src
COPY libs/platform/go.mod libs/platform/go.sum libs/platform/
COPY contracts/go.mod contracts/
COPY services/controle/go.mod services/controle/go.sum services/controle/
WORKDIR /src/services/controle
RUN go mod download
WORKDIR /src
COPY libs/platform/ libs/platform/
COPY contracts/ contracts/
COPY services/controle/ services/controle/
WORKDIR /src/services/controle
RUN go build -trimpath -ldflags="-s -w" -o /out/controle ./cmd/controle

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab
COPY --from=construction /out/controle /controle
USER 65532:65532
EXPOSE 9090
ENTRYPOINT ["/controle"]
