FROM node:22-alpine AS web-builder

WORKDIR /src/web

COPY web/package.json web/pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile

COPY web/ ./
RUN pnpm build

FROM golang:1.25-alpine AS go-builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/goroute ./cmd/goroute

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -S goroute && adduser -S -G goroute -h /home/goroute goroute

WORKDIR /app

COPY --from=go-builder /out/goroute /app/goroute
COPY --from=web-builder /src/web/dist /app/web/dist

RUN mkdir -p /home/goroute/.goroute && chown -R goroute:goroute /app /home/goroute

USER goroute

ENV HOME=/home/goroute
ENV GOROUTE_ENV=production

EXPOSE 2232

CMD ["/app/goroute"]
