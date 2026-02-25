
#!/bin/bas
deviceToken="0409e120b5b7944b35c36603c57ff95a891d6247bafc25db207947b0b5cc5dfe"


authKey="./AuthKey_RCFJ65XAT5.p8"
authKeyId=RCFJ65XAT5
teamId=G96QBG6QQJ
bundleId=io.kaller.Kaller-debug
endpoint=https://api.development.push.apple.com
#endpoint=https://api.push.apple.com


# --------------------------------------------------------------------------
read -r -d '' payload <<-'EOF'
{
   aps = {
      sound = "Note";
      badge = 1;
      category = "mycategory";
      alert = {
               title = "titllll";
               subtitle = "my subtitle";
               body = "my body text message";
         };
      data = {
        chatId = "dfb89860-d055-3e66-8466-1b6f01419e53";
        time = "1634817038584";
        user = "+972523702129#10";
        text = "Hey";
        type = "chat_message";
        attachments = {
            list = "";
            empty = true;
        };
         deletedBy = {
            list = "";
            empty = true;
        };
        replyTo = "";
        pushTitle = null;
        pushBody = null;
        msgId = "1c149780-3265-11ec-8daf-955b5901da5b";
        recipient = "+79850852550#10";
        socket_message_time = 1634817038592;
    };
      mutable-content = 1;
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
