#!/usr/bin/env sh
set -e
./pscert save --domain-name $DOMAIN_NAME --domain-email $DOMAIN_EMAIL --key-id alias/aws/ssm --pstore-path /allEnvs/$ENV_NAME/ssl
hostAddress=$(curl http://169.254.169.254/latest/meta-data/local-ipv4)
sed -i "s/REPlACE_HOSTADDRESS/$hostAddress/g" /etc/envoy/envoy.yaml
envoy -c /etc/envoy/envoy.yaml