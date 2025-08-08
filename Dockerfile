FROM node:20-alpine AS ui
WORKDIR /src
COPY ui ./ui
# Install deps and build UI into cmd/backend/static via vite.config.ts outDir
RUN cd ui && npm i && npm run build

FROM golang:1.24 AS build
WORKDIR /src
COPY . .
# Bring built static assets from UI stage
COPY --from=ui /src/cmd/backend/static ./cmd/backend/static
ENV GOTOOLCHAIN=auto
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/backend ./cmd/backend && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/agent ./cmd/agent

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=build /out/backend /bin/backend
COPY --from=build /out/agent /bin/agent
USER nonroot:nonroot
EXPOSE 8080 8090
ENTRYPOINT ["/bin/backend"]


