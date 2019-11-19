FROM golang as build-env
ARG TAG

WORKDIR /src
COPY . .
RUN make TAG=$TAG GOMOD="-mod=vendor" build

FROM registry.access.redhat.com/ubi7/ubi-minimal:latest AS final

ENV OPERATOR=/usr/local/bin/extendeddaemonset \
    USER_UID=1001 \
    USER_NAME=extendeddaemonset

# install operator binary
COPY --from=build-env /src/controller ${OPERATOR}

COPY --from=build-env /src/build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
