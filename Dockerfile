FROM golang:1.17-buster AS build

RUN groupadd -g 505 app && \
    useradd -u 505 -g 505 -k /etc/skel/ -s /bin/bash -m app

RUN mkdir /app && chown app:app /app
COPY --chown=app:app . /app/
WORKDIR /app/
ARG VERSION
RUN go mod tidy && go build -ldflags="-X main.Version=$VERSION" -o /app/bootstrap *.go

ENV LANG="en_US.utf-8" \
    LANGUAGE="en_US:en" \
    LC_ALL="en_US.utf-8"
FROM amazon/aws-cli:2.3.7

COPY --from=build --chown=app:app /app/bootstrap /app/bootstrap
ENTRYPOINT []
CMD [ "/app/bootstrap" ]
