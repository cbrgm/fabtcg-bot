FROM alpine:latest
LABEL maintainer="Christian Bargmann <chris@cbrgm.net>"

RUN apk add --update ca-certificates tini

COPY ./fabtcg-bot /usr/bin/fabtcg-bot

USER nobody
EXPOSE 8080
WORKDIR /fabtcg-bot

ENTRYPOINT ["/sbin/tini", "--"]
CMD [ "/usr/bin/fabtcg-bot", \
      "--log.level=debug", \
      "--http.addr=0.0.0.0:8080" ]
