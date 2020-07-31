FROM golang:1.14

RUN mkdir /src
COPY . /src
WORKDIR /src/cmd/operator

RUN go build -mod=readonly -o operator

ENTRYPOINT [ "./operator", "-cli"]
