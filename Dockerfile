FROM go:1.24-alpine

WORKDIR /app

COPY . .

RUN go mod tidy && go build -o app .

EXPOSE 1965

CMD ./app