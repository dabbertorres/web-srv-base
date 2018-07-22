#!/usr/bin/env sh

domain=$(cat /mail.conf)

echo "myhostname = mail.$domain" >> /etc/postfix/main.cf
echo "mydomain = $domain" >> /etc/postfix/main.cf
echo "smtp_tls_cert_file = /certs/$domain" >> /etc/postfix/main.cf

exec "$@"