#!/usr/bin/env bash

SCRIPT_PATH="$( cd -- "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"
DOMAIN=$1
COMMON_NAME=$1
SUBJECT="/C=CA/ST=None/L=NB/O=None/CN=$COMMON_NAME"
NUM_OF_DAYS=825
OUT="$SCRIPT_PATH/generated" # path to save generated data

if [ -z "$1" ]
then
  echo "Please supply a subdomain to create a certificate for";
  echo "e.g. www.mysite.com"
  exit;
fi

# Create a new private key if one doesnt exist, or use the xeisting one if it does
if [ -f $OUT/$DOMAIN.key ]; then
  KEY_OPT="-key"
else
  KEY_OPT="-keyout"
fi

# prepare dir
mkdir -p $OUT

# generate root certs
if [[ -f $OUT/rootCA.key && -f $OUT/rootCA.pem ]]; then
  echo "Both root key and certificate exist, reusing..."
else
  echo "Creating root key and certificate..."
  openssl genrsa -out $OUT/rootCA.key 2048
  openssl req -x509 -new -nodes -key $OUT/rootCA.key -sha256 -days 1024 -out $OUT/rootCA.pem
fi

# generate domain certs and sign
openssl req -new -newkey rsa:2048 -sha256 -nodes $KEY_OPT $OUT/$DOMAIN.key -subj "$SUBJECT" -out $OUT/device.csr
cat <<EOF > /tmp/__v3.ext
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = "$COMMON_NAME"
EOF
openssl x509 -req -in $OUT/device.csr -CA $OUT/rootCA.pem -CAkey $OUT/rootCA.key -CAcreateserial -out $OUT/device.crt -days $NUM_OF_DAYS -sha256 -extfile /tmp/__v3.ext

# move output files to final filenames
mv $OUT/device.csr "$OUT/$DOMAIN.csr"
cp $OUT/device.crt "$OUT/$DOMAIN.crt"

# remove temp file
rm -f $OUT/device.crt;

CACRT=$(cat $OUT/rootCA.pem | base64 -w0)
CAKEY=$(cat $OUT/rootCA.key  | base64 -w0)
TLSCRT=$(cat $OUT/$DOMAIN.crt | base64 -w0)
TLSKEY=$(cat $OUT/$DOMAIN.key  | base64 -w0)

echo "###########################[CA AUTH DATA]############################"
echo
echo "   ========> CA Root Cert [$OUT/rootCA.pem]"
echo -n "$CACRT"
echo
echo "   ========> CA Root Cert [$OUT/rootCA.key]"
echo -n "$CAKEY"
echo
echo "###########################[DOMAIN DATA]############################"
echo
echo "   ========> TLS Domain Cert [$OUT/rootCA.pem]"
echo -n "$TLSCRT"
echo
echo "   ========> TLS Domain Key  [$OUT/rootCA.key]"
echo -n "$TLSKEY"
echo