FROM alpine:latest
MAINTAINER Rohan Verma <hello@rohanverma.net>
RUN apk --no-cache add ca-certificates
WORKDIR /linkpage
COPY linkpage .
COPY config.sample.toml config.toml
CMD ["./linkpage"]
EXPOSE 8000

