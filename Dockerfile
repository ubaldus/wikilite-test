FROM golang:1.24-alpine

RUN apk add --no-cache git gcc g++ make musl-dev

WORKDIR /app

RUN git clone --recursive https://github.com/eja/wikilite.git .

RUN make

RUN chmod +x /app/wikilite

VOLUME /app/data

EXPOSE 35248

ENTRYPOINT ["./wikilite", "--db", "/app/data/wikilite.db", "--ai-model-path", "/app/data", "--web", "--web-host", "0.0.0.0"]
