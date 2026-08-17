# Версия pg_dump должна совпадать с версией сервера (db: postgres:18-alpine
# в docker-compose.yml), поэтому база тут та же.
FROM postgres:18-alpine

RUN apk add --no-cache rclone dcron

COPY backup.sh /usr/local/bin/backup.sh
RUN chmod +x /usr/local/bin/backup.sh

COPY crontab.txt /etc/crontabs/root

CMD ["crond", "-f", "-l", "2"]
