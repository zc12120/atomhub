FROM node:22 AS web-build
WORKDIR /app/web
COPY web/package*.json ./
RUN npm install
COPY web ./
RUN npm run build

FROM golang:1.26 AS go-build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-build /app/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -o /app/atomhub ./cmd/atomhub

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=go-build /app/atomhub /app/atomhub
COPY --from=go-build /app/web/dist /app/web/dist
ENV ATOMHUB_HTTP_ADDR=:8080
EXPOSE 8080
CMD ["/app/atomhub"]
