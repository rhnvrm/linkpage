FROM alpine:latest
LABEL org.opencontainers.image.authors="Rohan Verma <hello@rohanverma.net>"
RUN apk --no-cache add ca-certificates
WORKDIR /linkpage
ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/linkpage ./linkpage
COPY config.sample.toml config.toml
CMD ["./linkpage"]
