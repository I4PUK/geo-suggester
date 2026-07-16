FROM golang:1.20-alpine as build
COPY go.mod go.sum /src/
WORKDIR /src
RUN go mod download
COPY . /src/
RUN	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/srv ./cmd

FROM alpine:3.19 as production
COPY --from=build /bin/srv /app/srv
COPY pkg/template/search_template.json /app/search_template.json
COPY pkg/template/search_template_fuzzy.json /app/search_template_fuzzy.json
RUN apk --update add ca-certificates htop
ENV PORT 7784
EXPOSE $PORT
ENTRYPOINT /app/srv
