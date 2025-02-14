FROM playcourt/golang:1.23 AS builder
WORKDIR /app
COPY go.mod .
COPY go.sum .
COPY Makefile .
RUN make install
COPY . .
RUN make build

FROM playcourt/alpine:base
USER root
RUN apk add --no-cache curl
RUN addgroup -S nonroot \
    && adduser -S nonroot -G nonroot
USER nonroot

WORKDIR /app

ARG GITLAB_SECRET_TOKEN
ARG GROQ_API_KEY
ARG WEAVIATE_URL
ARG WEAVIATE_API_KEY
ARG PORT

RUN touch .env

RUN echo "GITLAB_SECRET_TOKEN=${GITLAB_SECRET_TOKEN}" >> .env
RUN echo "GROQ_API_KEY=${GROQ_API_KEY}" >> .env
RUN echo "WEAVIATE_URL=${WEAVIATE_URL}" >> .env
RUN echo "WEAVIATE_API_KEY=${WEAVIATE_API_KEY}" >> .env
RUN echo "PORT=${PORT}" >> .env

COPY --from=builder /app/main /app
CMD ["./main"]
