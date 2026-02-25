
#!/bin/bas
token=$1
deviceToken=token

authKey="./AuthKey_RCFJ65XAT5.p8"
authKeyId=RCFJ65XAT5
teamId=G96QBG6QQJ
bundleId=io.kaller.Kaller-debug

if [ $2 == prod ]
then
  endpoint=https://api.push.apple.com
else
  endpoint=https://api.development.push.apple.com
fi

# --------------------------------------------------------------------------
read -r -d '' payload <<-'EOF'
{
   aps = {
      badge = 1;
      category = "mycategory";
      alert = {
               title = "my title";
               subtitle = "my subtitle";
               body = "my body text message";
        };
         custom = {
          mykey = "myvalue";
      };
   };
}
EOF
# --------------------------------------------------------------------------

base64() {
   openssl base64 -e -A | tr -- '+/' '-_' | tr -d =
}

sign() {
   printf "$1" | openssl dgst -binary -sha256 -sign "$authKey" | base64
}

time=$(date +%s)
header=$(printf '{ "alg": "ES256", "kid": "%s" }' "$authKeyId" | base64)
claims=$(printf '{ "iss": "%s", "iat": %d }' "$teamId" "$time" | base64)
jwt="$header.$claims.$(sign $header.$claims)"

curl --verbose \
   --header "content-type: application/json" \
   --header "authorization: bearer $jwt" \
   --header "apns-topic: $bundleId" \
   --data "$payload" \
   $endpoint/3/device/$deviceToken
