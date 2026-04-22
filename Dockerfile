FROM golang:alpine AS builder

RUN apk add --no-cache --update git build-base
ENV CGO_ENABLED=0

WORKDIR /app

COPY . /app

WORKDIR /app/src
RUN go mod download
RUN go build -o /app/epusdt .

FROM alpine:latest AS runner
ENV TZ=Asia/Shanghai
RUN apk --no-cache add ca-certificates tzdata
ARG API_RATE_URL=""

WORKDIR /app
COPY --from=builder /app/src/static /app/static
COPY --from=builder /app/src/static /static
COPY --from=builder /app/src/.env.example /app/.env
RUN if [ -n "$API_RATE_URL" ]; then \
      sed -i "s|^api_rate_url=.*$|api_rate_url=${API_RATE_URL}|" /app/.env; \
    fi
COPY --from=builder /app/epusdt .

VOLUME /app/conf
ENTRYPOINT ["./epusdt", "http", "start"]
