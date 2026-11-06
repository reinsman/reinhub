FROM docker.example.com/ubuntu-cn:18.04
ARG TAG
ENV APP_VER=${TAG}
WORKDIR /app
COPY conf.toml examplesrv .
ENTRYPOINT ["/app/examplesrv"]