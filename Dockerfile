# مرحله ۱: ساخت فایل اجرایی
FROM golang:1.25-alpine AS builder
WORKDIR /app

# اول وابستگی‌ها، تا Docker از کش استفاده کنه
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /url-shortener .

# مرحله ۲: ایمیج نهایی، کوچیک و بدون ابزار ساخت
FROM alpine:3.20
RUN adduser -D -u 10001 appuser
USER appuser
COPY --from=builder /url-shortener /url-shortener
EXPOSE 8080
ENTRYPOINT ["/url-shortener"]