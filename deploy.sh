docker build -t server --platform linux/amd64  . \
  && docker tag server gcr.io/baileybutler-syd/sreekipedia \
  && docker push gcr.io/baileybutler-syd/sreekipedia
