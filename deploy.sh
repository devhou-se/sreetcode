docker build -t server --platform linux/amd64  . \
  && docker tag server gcr.io/baileybutler-syd/sreekipedia \
  && docker push gcr.io/baileybutler-syd/sreekipedia \
  && gcloud run deploy sreekipedia --image gcr.io/baileybutler-syd/sreekipedia:latest --platform managed --region asia-southeast1 --allow-unauthenticated
