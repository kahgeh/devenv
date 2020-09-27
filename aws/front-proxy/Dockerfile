FROM python:alpine3.6 as builder
WORKDIR /build
RUN apk add --update \
  python-dev &&\
  pip install --upgrade pip && \
  pip install Jinja2
RUN apk add curl
RUN apk del \
  python-dev \
  make && \
  rm -rf /var/cache/apk/*
RUN curl -fOL https://github.com/kahgeh/pscert/releases/download/v0.0.3/pscert_0.0.3_Linux_x86_64.tar.gz
RUN tar -xzf pscert_0.0.3_Linux_x86_64.tar.gz
ADD envoy-template.py .
ARG DOMAIN_NAME
ENV DOMAIN_NAME ${DOMAIN_NAME}
RUN python envoy-template.py

FROM envoyproxy/envoy-alpine:v1.15-latest
ARG DOMAIN_EMAIL
ENV DOMAIN_EMAIL ${DOMAIN_EMAIL}
ARG DOMAIN_NAME
ENV DOMAIN_NAME ${DOMAIN_NAME}
ARG ENV_NAME
ENV ENV_NAME ${ENV_NAME}
ENV ENVOY_UID 0
COPY --from=builder /build/envoy.yaml /etc/envoy
COPY --from=builder /build/pscert ./
RUN chmod a+rw -R /etc/envoy
RUN chmod a+rw /etc/envoy/envoy.yaml
RUN chmod a+x ./pscert
ADD init-and-run.sh .
RUN chmod a+x ./init-and-run.sh
RUN mkdir -p /etc/letsencrypt/live
RUN chmod a+rw /etc/letsencrypt/live
RUN apk add --no-cache certbot
RUN apk add --no-cache curl
EXPOSE 80/tcp
EXPOSE 443/tcp
EXPOSE 9901/tcp
CMD ./init-and-run.sh
