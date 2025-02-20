FROM golang:1.23.4-alpine

RUN apk add --no-cache git gcc g++ make musl-dev

WORKDIR /app

RUN git clone https://github.com/eja/wikilite.git .

RUN make

RUN chmod +x /app/wikilite

VOLUME /app/data

EXPOSE 35248

ENTRYPOINT ["./wikilite", "--db", "/app/data/wikilite.db", "--web", "--web-host", "0.0.0.0"]
