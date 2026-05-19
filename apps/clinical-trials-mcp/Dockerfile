FROM golang:1.25-bookworm AS build

WORKDIR /src

COPY ./apps/clinical-trials-mcp/go.mod ./apps/clinical-trials-mcp/go.sum ./
RUN go mod download

COPY ./apps/clinical-trials-mcp ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /clinical-trials-mcp .

FROM gcr.io/distroless/static-debian12

ENV MCP_HOST=0.0.0.0
ENV MCP_PATH=/mcp
ENV CT_API_BASE_URL=https://clinicaltrials.gov/api/v2
ENV EU_CT_API_BASE_URL=https://euclinicaltrials.eu/ctis-public-api
ENV CT_REQUEST_TIMEOUT_SECONDS=30
ENV CT_MAX_PAGE_SIZE=25

EXPOSE 8080

COPY --from=build /clinical-trials-mcp /clinical-trials-mcp

CMD ["/clinical-trials-mcp"]
