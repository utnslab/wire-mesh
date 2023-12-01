FROM golang:1.20.5-alpine3.18 as build

WORKDIR /go/src/server

COPY main.go .
RUN go mod init server && \
  go mod tidy && \
  go build -o /server main.go

FROM gcr.io/distroless/base-debian10

COPY --from=build /server /

EXPOSE 50051

CMD ["/server"]